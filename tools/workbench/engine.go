package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/models"
)

// DefaultPageSize is the default page size for fetching.
const DefaultPageSize = 25

// extractPayloadKey extracts a short identifier from a JSON payload for use as a state key.
// It tries to use the "Reference" field if present, otherwise uses a truncated version.
func extractPayloadKey(payload json.RawMessage) string {
	if len(payload) == 0 {
		return ""
	}
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err == nil {
		if ref, ok := data["Reference"].(string); ok && ref != "" {
			// Use just the reference, truncate if too long
			if len(ref) > 30 {
				return ref[:30]
			}
			return ref
		}
	}
	// Fallback: use truncated payload
	s := string(payload)
	if len(s) > 20 {
		return s[:20] + "..."
	}
	return s
}

// Engine is a lightweight execution engine that replaces Temporal workflows
// with direct function calls for local development.
type Engine struct {
	connectorID models.ConnectorID
	plugin      models.Plugin
	storage     *MemoryStorage
	debug       *DebugStore
	tasks       *TaskTracker
	logger      logging.Logger

	mu sync.RWMutex

	// Execution state
	installed   bool
	tasksTree   *models.ConnectorTasksTree
	pageSize    int
	running     bool

	// Fetch state (for step-by-step execution)
	accountsFetchState   *fetchState
	paymentsFetchState   map[string]*fetchState // keyed by fromPayload ID
	balancesFetchState   map[string]*fetchState
	externalAccountsState map[string]*fetchState
	othersFetchState     map[string]map[string]*fetchState // keyed by name, then fromPayload ID
}

type fetchState struct {
	State    json.RawMessage
	HasMore  bool
	PagesFetched int
	TotalItems   int
}

// NewEngine creates a new dev engine.
func NewEngine(
	connectorID models.ConnectorID,
	plugin models.Plugin,
	storage *MemoryStorage,
	debug *DebugStore,
	tasks *TaskTracker,
	logger logging.Logger,
) *Engine {
	return &Engine{
		connectorID:          connectorID,
		plugin:               plugin,
		storage:              storage,
		debug:                debug,
		tasks:                tasks,
		logger:               logger,
		pageSize:             DefaultPageSize,
		paymentsFetchState:   make(map[string]*fetchState),
		balancesFetchState:   make(map[string]*fetchState),
		externalAccountsState: make(map[string]*fetchState),
		othersFetchState:     make(map[string]map[string]*fetchState),
	}
}

// SetPageSize sets the page size for fetching.
func (e *Engine) SetPageSize(size int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.pageSize = size
}

// Install installs the connector.
func (e *Engine) Install(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.debug.Log("install", "Installing connector")

	callID := e.debug.LogPluginCall("Install", models.InstallRequest{
		ConnectorID: e.connectorID.String(),
	})
	start := time.Now()

	resp, err := e.plugin.Install(ctx, models.InstallRequest{
		ConnectorID: e.connectorID.String(),
	})

	e.debug.LogPluginResult(callID, resp, time.Since(start), err)

	if err != nil {
		e.debug.LogError("install", err)
		return fmt.Errorf("install failed: %w", err)
	}

	e.tasksTree = &resp.Workflow
	e.storage.SetTasksTree(e.tasksTree)
	e.installed = true

	// Set up task tracker with the workflow tree
	if e.tasks != nil {
		e.tasks.SetTaskTree(resp.Workflow)
	}

	e.debug.Log("install", fmt.Sprintf("Connector installed with %d root tasks", len(resp.Workflow)))
	e.logger.Infof("Connector installed with workflow tree: %d root tasks", len(resp.Workflow))

	return nil
}

// Uninstall uninstalls the connector.
func (e *Engine) Uninstall(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.installed {
		return nil
	}

	e.debug.Log("uninstall", "Uninstalling connector")

	callID := e.debug.LogPluginCall("Uninstall", models.UninstallRequest{
		ConnectorID:    e.connectorID.String(),
		WebhookConfigs: e.storage.GetWebhookConfigs(),
	})
	start := time.Now()

	_, err := e.plugin.Uninstall(ctx, models.UninstallRequest{
		ConnectorID:    e.connectorID.String(),
		WebhookConfigs: e.storage.GetWebhookConfigs(),
	})

	e.debug.LogPluginResult(callID, nil, time.Since(start), err)

	if err != nil {
		e.debug.LogError("uninstall", err)
		return fmt.Errorf("uninstall failed: %w", err)
	}

	e.installed = false
	e.debug.Log("uninstall", "Connector uninstalled")

	return nil
}

// RunOneCycle runs one complete fetch cycle through the workflow tree.
func (e *Engine) RunOneCycle(ctx context.Context) error {
	e.mu.Lock()
	if !e.installed {
		e.mu.Unlock()
		return fmt.Errorf("connector not installed")
	}
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("cycle already running")
	}
	e.running = true
	tree := e.tasksTree
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.running = false
		e.mu.Unlock()
	}()

	if tree == nil || len(*tree) == 0 {
		return nil
	}

	e.debug.Log("cycle", "Starting fetch cycle")

	// Execute the workflow tree
	for _, task := range *tree {
		if err := e.executeTask(ctx, task, nil); err != nil {
			e.debug.LogError("cycle", err)
			return err
		}
	}

	e.debug.Log("cycle", "Fetch cycle complete")
	return nil
}

// IsRunning returns whether a cycle is currently running.
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// executeTask executes a single task from the workflow tree.
func (e *Engine) executeTask(ctx context.Context, task models.ConnectorTaskTree, fromPayload json.RawMessage) error {
	switch task.TaskType {
	case models.TASK_FETCH_ACCOUNTS:
		return e.executeFetchAccountsTask(ctx, task, fromPayload)
	case models.TASK_FETCH_PAYMENTS:
		return e.executeFetchPaymentsTask(ctx, task, fromPayload)
	case models.TASK_FETCH_BALANCES:
		return e.executeFetchBalancesTask(ctx, task, fromPayload)
	case models.TASK_FETCH_EXTERNAL_ACCOUNTS:
		return e.executeFetchExternalAccountsTask(ctx, task, fromPayload)
	case models.TASK_FETCH_OTHERS:
		return e.executeFetchOthersTask(ctx, task, fromPayload)
	case models.TASK_CREATE_WEBHOOKS:
		return e.executeCreateWebhooksTask(ctx, task, fromPayload)
	default:
		e.debug.Log("task", fmt.Sprintf("Unknown task type: %s", task.TaskType))
		return nil
	}
}

func (e *Engine) executeFetchAccountsTask(ctx context.Context, task models.ConnectorTaskTree, fromPayload json.RawMessage) error {
	e.debug.Log("fetch_accounts", "Fetching accounts...")

	// Fetch all pages
	var state json.RawMessage
	hasMore := true
	totalFetched := 0
	pageNum := 0

	for hasMore {
		pageNum++
		
		// Start task tracking
		var exec *TaskExecution
		if e.tasks != nil {
			exec = e.tasks.StartTask(models.TASK_FETCH_ACCOUNTS, "", fromPayload)
			exec.PageNumber = pageNum
		}

		// Wait for step signal if in step mode
		if e.tasks != nil && !e.tasks.WaitForStep() {
			return fmt.Errorf("execution stopped")
		}

		callID := e.debug.LogPluginCall("FetchNextAccounts", models.FetchNextAccountsRequest{
			FromPayload: fromPayload,
			State:       state,
			PageSize:    e.pageSize,
		})
		start := time.Now()

		resp, err := e.plugin.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{
			FromPayload: fromPayload,
			State:       state,
			PageSize:    e.pageSize,
		})

		e.debug.LogPluginResult(callID, resp, time.Since(start), err)

		// Complete task tracking
		if e.tasks != nil && exec != nil {
			itemCount := 0
			if err == nil {
				itemCount = len(resp.Accounts)
			}
			e.tasks.CompleteTask(exec, itemCount, resp.HasMore, err)
		}

		if err != nil {
			e.debug.LogError("fetch_accounts", err)
			return fmt.Errorf("fetch accounts failed: %w", err)
		}

		if len(resp.Accounts) > 0 {
			e.storage.StoreAccounts(resp.Accounts)
			totalFetched += len(resp.Accounts)
		}

		// Track state change
		if state != nil || resp.NewState != nil {
			e.debug.LogStateChange("accounts", state, resp.NewState)
		}

		state = resp.NewState
		hasMore = resp.HasMore

		// Execute child tasks for each account
		for _, account := range resp.Accounts {
			accountPayload, _ := json.Marshal(account)
			for _, childTask := range task.NextTasks {
				// Track child task in tree
				if e.tasks != nil {
					name := account.Reference
					if account.Name != nil {
						name = *account.Name
					}
					e.tasks.AddChildTask(models.TASK_FETCH_ACCOUNTS, childTask.TaskType, name, accountPayload)
				}
				if err := e.executeTask(ctx, childTask, accountPayload); err != nil {
					return err
				}
			}
		}
	}

	e.debug.Log("fetch_accounts", fmt.Sprintf("Fetched %d accounts", totalFetched))
	e.storage.SaveState("accounts", state)

	return nil
}

func (e *Engine) executeFetchPaymentsTask(ctx context.Context, task models.ConnectorTaskTree, fromPayload json.RawMessage) error {
	e.debug.Log("fetch_payments", "Fetching payments...")

	var state json.RawMessage
	hasMore := true
	totalFetched := 0
	pageNum := 0

	for hasMore {
		pageNum++

		// Start task tracking
		var exec *TaskExecution
		if e.tasks != nil {
			exec = e.tasks.StartTask(models.TASK_FETCH_PAYMENTS, "", fromPayload)
			exec.PageNumber = pageNum
		}

		// Wait for step signal if in step mode
		if e.tasks != nil && !e.tasks.WaitForStep() {
			return fmt.Errorf("execution stopped")
		}

		callID := e.debug.LogPluginCall("FetchNextPayments", models.FetchNextPaymentsRequest{
			FromPayload: fromPayload,
			State:       state,
			PageSize:    e.pageSize,
		})
		start := time.Now()

		resp, err := e.plugin.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{
			FromPayload: fromPayload,
			State:       state,
			PageSize:    e.pageSize,
		})

		e.debug.LogPluginResult(callID, resp, time.Since(start), err)

		// Complete task tracking
		if e.tasks != nil && exec != nil {
			itemCount := 0
			if err == nil {
				itemCount = len(resp.Payments)
			}
			e.tasks.CompleteTask(exec, itemCount, resp.HasMore, err)
		}

		if err != nil {
			e.debug.LogError("fetch_payments", err)
			return fmt.Errorf("fetch payments failed: %w", err)
		}

		if len(resp.Payments) > 0 {
			e.storage.StorePayments(resp.Payments)
			totalFetched += len(resp.Payments)
		}

		if state != nil || resp.NewState != nil {
			e.debug.LogStateChange("payments", state, resp.NewState)
		}

		state = resp.NewState
		hasMore = resp.HasMore
	}

	e.debug.Log("fetch_payments", fmt.Sprintf("Fetched %d payments", totalFetched))

	// Save state for UI display
	stateKey := "payments"
	if payloadKey := extractPayloadKey(fromPayload); payloadKey != "" {
		stateKey = fmt.Sprintf("payments:%s", payloadKey)
	}
	e.storage.SaveState(stateKey, state)

	return nil
}

func (e *Engine) executeFetchBalancesTask(ctx context.Context, task models.ConnectorTaskTree, fromPayload json.RawMessage) error {
	e.debug.Log("fetch_balances", "Fetching balances...")

	var state json.RawMessage
	hasMore := true
	totalFetched := 0
	pageNum := 0

	for hasMore {
		pageNum++

		// Start task tracking
		var exec *TaskExecution
		if e.tasks != nil {
			exec = e.tasks.StartTask(models.TASK_FETCH_BALANCES, "", fromPayload)
			exec.PageNumber = pageNum
		}

		// Wait for step signal if in step mode
		if e.tasks != nil && !e.tasks.WaitForStep() {
			return fmt.Errorf("execution stopped")
		}

		callID := e.debug.LogPluginCall("FetchNextBalances", models.FetchNextBalancesRequest{
			FromPayload: fromPayload,
			State:       state,
			PageSize:    e.pageSize,
		})
		start := time.Now()

		resp, err := e.plugin.FetchNextBalances(ctx, models.FetchNextBalancesRequest{
			FromPayload: fromPayload,
			State:       state,
			PageSize:    e.pageSize,
		})

		e.debug.LogPluginResult(callID, resp, time.Since(start), err)

		// Complete task tracking
		if e.tasks != nil && exec != nil {
			itemCount := 0
			if err == nil {
				itemCount = len(resp.Balances)
			}
			e.tasks.CompleteTask(exec, itemCount, resp.HasMore, err)
		}

		if err != nil {
			e.debug.LogError("fetch_balances", err)
			return fmt.Errorf("fetch balances failed: %w", err)
		}

		if len(resp.Balances) > 0 {
			e.storage.StoreBalances(resp.Balances)
			totalFetched += len(resp.Balances)
		}

		if state != nil || resp.NewState != nil {
			e.debug.LogStateChange("balances", state, resp.NewState)
		}

		state = resp.NewState
		hasMore = resp.HasMore
	}

	e.debug.Log("fetch_balances", fmt.Sprintf("Fetched %d balances", totalFetched))

	// Save state for UI display
	stateKey := "balances"
	if payloadKey := extractPayloadKey(fromPayload); payloadKey != "" {
		stateKey = fmt.Sprintf("balances:%s", payloadKey)
	}
	e.storage.SaveState(stateKey, state)

	return nil
}

func (e *Engine) executeFetchExternalAccountsTask(ctx context.Context, task models.ConnectorTaskTree, fromPayload json.RawMessage) error {
	e.debug.Log("fetch_external_accounts", "Fetching external accounts...")

	var state json.RawMessage
	hasMore := true
	totalFetched := 0
	pageNum := 0

	for hasMore {
		pageNum++

		// Start task tracking
		var exec *TaskExecution
		if e.tasks != nil {
			exec = e.tasks.StartTask(models.TASK_FETCH_EXTERNAL_ACCOUNTS, "", fromPayload)
			exec.PageNumber = pageNum
		}

		// Wait for step signal if in step mode
		if e.tasks != nil && !e.tasks.WaitForStep() {
			return fmt.Errorf("execution stopped")
		}

		callID := e.debug.LogPluginCall("FetchNextExternalAccounts", models.FetchNextExternalAccountsRequest{
			FromPayload: fromPayload,
			State:       state,
			PageSize:    e.pageSize,
		})
		start := time.Now()

		resp, err := e.plugin.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{
			FromPayload: fromPayload,
			State:       state,
			PageSize:    e.pageSize,
		})

		e.debug.LogPluginResult(callID, resp, time.Since(start), err)

		// Complete task tracking
		if e.tasks != nil && exec != nil {
			itemCount := 0
			if err == nil {
				itemCount = len(resp.ExternalAccounts)
			}
			e.tasks.CompleteTask(exec, itemCount, resp.HasMore, err)
		}

		if err != nil {
			e.debug.LogError("fetch_external_accounts", err)
			return fmt.Errorf("fetch external accounts failed: %w", err)
		}

		if len(resp.ExternalAccounts) > 0 {
			e.storage.StoreExternalAccounts(resp.ExternalAccounts)
			totalFetched += len(resp.ExternalAccounts)
		}

		if state != nil || resp.NewState != nil {
			e.debug.LogStateChange("external_accounts", state, resp.NewState)
		}

		state = resp.NewState
		hasMore = resp.HasMore
	}

	e.debug.Log("fetch_external_accounts", fmt.Sprintf("Fetched %d external accounts", totalFetched))

	// Save state for UI display
	stateKey := "external_accounts"
	if payloadKey := extractPayloadKey(fromPayload); payloadKey != "" {
		stateKey = fmt.Sprintf("external_accounts:%s", payloadKey)
	}
	e.storage.SaveState(stateKey, state)

	return nil
}

func (e *Engine) executeFetchOthersTask(ctx context.Context, task models.ConnectorTaskTree, fromPayload json.RawMessage) error {
	name := task.Name
	if name == "" {
		name = "default"
	}

	e.debug.Log("fetch_others", fmt.Sprintf("Fetching others (%s)...", name))

	var state json.RawMessage
	hasMore := true
	totalFetched := 0
	pageNum := 0

	for hasMore {
		pageNum++

		// Start task tracking
		var exec *TaskExecution
		if e.tasks != nil {
			exec = e.tasks.StartTask(models.TASK_FETCH_OTHERS, name, fromPayload)
			exec.PageNumber = pageNum
		}

		// Wait for step signal if in step mode
		if e.tasks != nil && !e.tasks.WaitForStep() {
			return fmt.Errorf("execution stopped")
		}

		callID := e.debug.LogPluginCall("FetchNextOthers", models.FetchNextOthersRequest{
			Name:        name,
			FromPayload: fromPayload,
			State:       state,
			PageSize:    e.pageSize,
		})
		start := time.Now()

		resp, err := e.plugin.FetchNextOthers(ctx, models.FetchNextOthersRequest{
			Name:        name,
			FromPayload: fromPayload,
			State:       state,
			PageSize:    e.pageSize,
		})

		e.debug.LogPluginResult(callID, resp, time.Since(start), err)

		// Complete task tracking
		if e.tasks != nil && exec != nil {
			itemCount := 0
			if err == nil {
				itemCount = len(resp.Others)
			}
			e.tasks.CompleteTask(exec, itemCount, resp.HasMore, err)
		}

		if err != nil {
			e.debug.LogError("fetch_others", err)
			return fmt.Errorf("fetch others (%s) failed: %w", name, err)
		}

		if len(resp.Others) > 0 {
			e.storage.StoreOthers(name, resp.Others)
			totalFetched += len(resp.Others)

			// Execute child tasks for each other item
			for _, other := range resp.Others {
				otherPayload, _ := json.Marshal(other)
				for _, childTask := range task.NextTasks {
					if err := e.executeTask(ctx, childTask, otherPayload); err != nil {
						return err
					}
				}
			}
		}

		if state != nil || resp.NewState != nil {
			e.debug.LogStateChange("others_"+name, state, resp.NewState)
		}

		state = resp.NewState
		hasMore = resp.HasMore
	}

	e.debug.Log("fetch_others", fmt.Sprintf("Fetched %d others (%s)", totalFetched, name))

	// Save state for UI display
	stateKey := fmt.Sprintf("others:%s", name)
	if payloadKey := extractPayloadKey(fromPayload); payloadKey != "" {
		stateKey = fmt.Sprintf("others:%s:%s", name, payloadKey)
	}
	e.storage.SaveState(stateKey, state)

	return nil
}

func (e *Engine) executeCreateWebhooksTask(ctx context.Context, task models.ConnectorTaskTree, fromPayload json.RawMessage) error {
	e.debug.Log("create_webhooks", "Creating webhooks...")

	// Start task tracking
	var exec *TaskExecution
	if e.tasks != nil {
		exec = e.tasks.StartTask(models.TASK_CREATE_WEBHOOKS, "", fromPayload)
		exec.PageNumber = 1
	}

	// Wait for step signal if in step mode
	if e.tasks != nil && !e.tasks.WaitForStep() {
		return fmt.Errorf("execution stopped")
	}

	callID := e.debug.LogPluginCall("CreateWebhooks", models.CreateWebhooksRequest{
		FromPayload:    fromPayload,
		ConnectorID:    e.connectorID.String(),
		WebhookBaseUrl: "http://localhost:8080/webhooks",
	})
	start := time.Now()

	resp, err := e.plugin.CreateWebhooks(ctx, models.CreateWebhooksRequest{
		FromPayload:    fromPayload,
		ConnectorID:    e.connectorID.String(),
		WebhookBaseUrl: "http://localhost:8080/webhooks",
	})

	e.debug.LogPluginResult(callID, resp, time.Since(start), err)

	// Complete task tracking
	if e.tasks != nil && exec != nil {
		itemCount := 0
		if err == nil {
			itemCount = len(resp.Configs)
		}
		e.tasks.CompleteTask(exec, itemCount, false, err)
	}

	if err != nil {
		e.debug.LogError("create_webhooks", err)
		return fmt.Errorf("create webhooks failed: %w", err)
	}

	if len(resp.Configs) > 0 {
		e.storage.SetWebhookConfigs(resp.Configs)
	}

	e.debug.Log("create_webhooks", fmt.Sprintf("Created %d webhook configs", len(resp.Configs)))

	return nil
}

// === Manual Operations ===

// FetchAccountsOnePage fetches one page of accounts.
func (e *Engine) FetchAccountsOnePage(ctx context.Context, fromPayload json.RawMessage) (*models.FetchNextAccountsResponse, error) {
	e.mu.Lock()
	state := e.accountsFetchState
	if state == nil {
		state = &fetchState{HasMore: true}
		e.accountsFetchState = state
	}
	currentState := state.State
	e.mu.Unlock()

	if !state.HasMore {
		return &models.FetchNextAccountsResponse{HasMore: false}, nil
	}

	callID := e.debug.LogPluginCall("FetchNextAccounts", models.FetchNextAccountsRequest{
		FromPayload: fromPayload,
		State:       currentState,
		PageSize:    e.pageSize,
	})
	start := time.Now()

	resp, err := e.plugin.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{
		FromPayload: fromPayload,
		State:       currentState,
		PageSize:    e.pageSize,
	})

	e.debug.LogPluginResult(callID, resp, time.Since(start), err)

	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	state.State = resp.NewState
	state.HasMore = resp.HasMore
	state.PagesFetched++
	state.TotalItems += len(resp.Accounts)
	e.mu.Unlock()

	if len(resp.Accounts) > 0 {
		e.storage.StoreAccounts(resp.Accounts)
	}

	return &resp, nil
}

// FetchPaymentsOnePage fetches one page of payments.
func (e *Engine) FetchPaymentsOnePage(ctx context.Context, fromPayload json.RawMessage) (*models.FetchNextPaymentsResponse, error) {
	key := string(fromPayload)
	if key == "" {
		key = "_root"
	}

	e.mu.Lock()
	state, ok := e.paymentsFetchState[key]
	if !ok {
		state = &fetchState{HasMore: true}
		e.paymentsFetchState[key] = state
	}
	currentState := state.State
	e.mu.Unlock()

	if !state.HasMore {
		return &models.FetchNextPaymentsResponse{HasMore: false}, nil
	}

	callID := e.debug.LogPluginCall("FetchNextPayments", models.FetchNextPaymentsRequest{
		FromPayload: fromPayload,
		State:       currentState,
		PageSize:    e.pageSize,
	})
	start := time.Now()

	resp, err := e.plugin.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{
		FromPayload: fromPayload,
		State:       currentState,
		PageSize:    e.pageSize,
	})

	e.debug.LogPluginResult(callID, resp, time.Since(start), err)

	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	state.State = resp.NewState
	state.HasMore = resp.HasMore
	state.PagesFetched++
	state.TotalItems += len(resp.Payments)
	e.mu.Unlock()

	if len(resp.Payments) > 0 {
		e.storage.StorePayments(resp.Payments)
	}

	return &resp, nil
}

// FetchBalancesOnePage fetches one page of balances.
func (e *Engine) FetchBalancesOnePage(ctx context.Context, fromPayload json.RawMessage) (*models.FetchNextBalancesResponse, error) {
	key := string(fromPayload)
	if key == "" {
		key = "_root"
	}

	e.mu.Lock()
	state, ok := e.balancesFetchState[key]
	if !ok {
		state = &fetchState{HasMore: true}
		e.balancesFetchState[key] = state
	}
	currentState := state.State
	e.mu.Unlock()

	if !state.HasMore {
		return &models.FetchNextBalancesResponse{HasMore: false}, nil
	}

	callID := e.debug.LogPluginCall("FetchNextBalances", models.FetchNextBalancesRequest{
		FromPayload: fromPayload,
		State:       currentState,
		PageSize:    e.pageSize,
	})
	start := time.Now()

	resp, err := e.plugin.FetchNextBalances(ctx, models.FetchNextBalancesRequest{
		FromPayload: fromPayload,
		State:       currentState,
		PageSize:    e.pageSize,
	})

	e.debug.LogPluginResult(callID, resp, time.Since(start), err)

	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	state.State = resp.NewState
	state.HasMore = resp.HasMore
	state.PagesFetched++
	state.TotalItems += len(resp.Balances)
	e.mu.Unlock()

	if len(resp.Balances) > 0 {
		e.storage.StoreBalances(resp.Balances)
	}

	return &resp, nil
}

// FetchExternalAccountsOnePage fetches one page of external accounts.
func (e *Engine) FetchExternalAccountsOnePage(ctx context.Context, fromPayload json.RawMessage) (*models.FetchNextExternalAccountsResponse, error) {
	key := string(fromPayload)
	if key == "" {
		key = "_root"
	}

	e.mu.Lock()
	state, ok := e.externalAccountsState[key]
	if !ok {
		state = &fetchState{HasMore: true}
		e.externalAccountsState[key] = state
	}
	currentState := state.State
	e.mu.Unlock()

	if !state.HasMore {
		return &models.FetchNextExternalAccountsResponse{HasMore: false}, nil
	}

	callID := e.debug.LogPluginCall("FetchNextExternalAccounts", models.FetchNextExternalAccountsRequest{
		FromPayload: fromPayload,
		State:       currentState,
		PageSize:    e.pageSize,
	})
	start := time.Now()

	resp, err := e.plugin.FetchNextExternalAccounts(ctx, models.FetchNextExternalAccountsRequest{
		FromPayload: fromPayload,
		State:       currentState,
		PageSize:    e.pageSize,
	})

	e.debug.LogPluginResult(callID, resp, time.Since(start), err)

	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	state.State = resp.NewState
	state.HasMore = resp.HasMore
	state.PagesFetched++
	state.TotalItems += len(resp.ExternalAccounts)
	e.mu.Unlock()

	if len(resp.ExternalAccounts) > 0 {
		e.storage.StoreExternalAccounts(resp.ExternalAccounts)
	}

	return &resp, nil
}

// CreateTransfer creates a transfer.
func (e *Engine) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (*models.CreateTransferResponse, error) {
	callID := e.debug.LogPluginCall("CreateTransfer", req)
	start := time.Now()

	resp, err := e.plugin.CreateTransfer(ctx, req)

	e.debug.LogPluginResult(callID, resp, time.Since(start), err)

	return &resp, err
}

// CreatePayout creates a payout.
func (e *Engine) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (*models.CreatePayoutResponse, error) {
	callID := e.debug.LogPluginCall("CreatePayout", req)
	start := time.Now()

	resp, err := e.plugin.CreatePayout(ctx, req)

	e.debug.LogPluginResult(callID, resp, time.Since(start), err)

	return &resp, err
}

// ResetFetchState resets all fetch states for re-fetching from scratch.
func (e *Engine) ResetFetchState() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.accountsFetchState = nil
	e.paymentsFetchState = make(map[string]*fetchState)
	e.balancesFetchState = make(map[string]*fetchState)
	e.externalAccountsState = make(map[string]*fetchState)
	e.othersFetchState = make(map[string]map[string]*fetchState)

	e.storage.ClearAllStates()
	e.debug.Log("reset", "Fetch state reset")
}

// GetFetchStatus returns the current fetch status.
type FetchStatus struct {
	Accounts struct {
		HasMore      bool `json:"has_more"`
		PagesFetched int  `json:"pages_fetched"`
		TotalItems   int  `json:"total_items"`
	} `json:"accounts"`
	Payments map[string]struct {
		HasMore      bool `json:"has_more"`
		PagesFetched int  `json:"pages_fetched"`
		TotalItems   int  `json:"total_items"`
	} `json:"payments"`
	Balances map[string]struct {
		HasMore      bool `json:"has_more"`
		PagesFetched int  `json:"pages_fetched"`
		TotalItems   int  `json:"total_items"`
	} `json:"balances"`
}

func (e *Engine) GetFetchStatus() FetchStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status := FetchStatus{
		Payments: make(map[string]struct {
			HasMore      bool `json:"has_more"`
			PagesFetched int  `json:"pages_fetched"`
			TotalItems   int  `json:"total_items"`
		}),
		Balances: make(map[string]struct {
			HasMore      bool `json:"has_more"`
			PagesFetched int  `json:"pages_fetched"`
			TotalItems   int  `json:"total_items"`
		}),
	}

	if e.accountsFetchState != nil {
		status.Accounts.HasMore = e.accountsFetchState.HasMore
		status.Accounts.PagesFetched = e.accountsFetchState.PagesFetched
		status.Accounts.TotalItems = e.accountsFetchState.TotalItems
	} else {
		status.Accounts.HasMore = true
	}

	for k, v := range e.paymentsFetchState {
		status.Payments[k] = struct {
			HasMore      bool `json:"has_more"`
			PagesFetched int  `json:"pages_fetched"`
			TotalItems   int  `json:"total_items"`
		}{
			HasMore:      v.HasMore,
			PagesFetched: v.PagesFetched,
			TotalItems:   v.TotalItems,
		}
	}

	for k, v := range e.balancesFetchState {
		status.Balances[k] = struct {
			HasMore      bool `json:"has_more"`
			PagesFetched int  `json:"pages_fetched"`
			TotalItems   int  `json:"total_items"`
		}{
			HasMore:      v.HasMore,
			PagesFetched: v.PagesFetched,
			TotalItems:   v.TotalItems,
		}
	}

	return status
}
