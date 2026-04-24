package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/spf13/cobra"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"
)

// knownCapabilities maps capability strings to their task type and workflow name.
// Ordered longest-first so parseScheduleID matches FETCH_EXTERNAL_ACCOUNTS before FETCH_ACCOUNTS.
var knownCapabilities = []struct {
	name         string
	capability   models.Capability
	taskType     models.TaskType
	workflowName string
}{
	{"FETCH_EXTERNAL_ACCOUNTS", models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS, models.TASK_FETCH_EXTERNAL_ACCOUNTS, workflow.RunFetchNextExternalAccounts},
	{"FETCH_ACCOUNTS", models.CAPABILITY_FETCH_ACCOUNTS, models.TASK_FETCH_ACCOUNTS, workflow.RunFetchNextAccounts},
	{"FETCH_BALANCES", models.CAPABILITY_FETCH_BALANCES, models.TASK_FETCH_BALANCES, workflow.RunFetchNextBalances},
	{"FETCH_PAYMENTS", models.CAPABILITY_FETCH_PAYMENTS, models.TASK_FETCH_PAYMENTS, workflow.RunFetchNextPayments},
	{"FETCH_OTHERS", models.CAPABILITY_FETCH_OTHERS, models.TASK_FETCH_OTHERS, workflow.RunFetchNextOthers},
	{"CREATE_WEBHOOKS", models.CAPABILITY_CREATE_WEBHOOKS, models.TASK_CREATE_WEBHOOKS, workflow.RunCreateWebhooks},
}

func newRecreateSchedules() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "recreate-schedules",
		Short:        "Recreate missing Temporal schedules for active connectors",
		SilenceUsage: true,
		RunE:         runRecreateSchedules(),
	}
	commonFlags(cmd)
	return cmd
}

func runRecreateSchedules() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		setLogger()

		stack, _ := cmd.Flags().GetString(StackFlag)
		logger := logging.NewDefaultLogger(cmd.OutOrStdout(), true, true, false)

		var rs *RecreateSchedules
		commonOpts, err := commonOptions(cmd)
		if err != nil {
			return fmt.Errorf("failed to configure common options: %w", err)
		}

		options := []fx.Option{
			fx.Supply(fx.Annotate(logger, fx.As(new(logging.Logger)))),
			commonOpts,
			fx.Provide(func(logger logging.Logger, temporalClient client.Client, storage storage.Storage) *RecreateSchedules {
				return NewRecreateSchedules(logger, temporalClient, storage, stack)
			}),
			fx.Populate(&rs),
		}

		app := fx.New(options...)
		if err := app.Start(cmd.Context()); err != nil {
			return err
		}
		defer func() {
			if err := app.Stop(context.Background()); err != nil {
				logger.Errorf("failed to stop app: %s", err)
			}
		}()

		return rs.Run(cmd.Context())
	}
}

// scheduleClientProvider is the subset of client.Client used by RecreateSchedules.
type scheduleClientProvider interface {
	ScheduleClient() client.ScheduleClient
}

// RecreateSchedules recreates missing Temporal schedules for active connectors
// by reading their stored task trees, schedules, and account data from the database.
type RecreateSchedules struct {
	logger         logging.Logger
	temporalClient scheduleClientProvider
	storage        storage.Storage
	stack          string
}

// NewRecreateSchedules creates a new RecreateSchedules instance with the given dependencies.
func NewRecreateSchedules(logger logging.Logger, temporalClient scheduleClientProvider, storage storage.Storage, stack string) *RecreateSchedules {
	return &RecreateSchedules{
		logger:         logger,
		temporalClient: temporalClient,
		storage:        storage,
		stack:          stack,
	}
}

// Run iterates over all active connectors and recreates their Temporal polling
// schedules (both root and sub-schedules). Existing schedules are silently skipped (idempotent).
func (r *RecreateSchedules) Run(ctx context.Context) error {
	r.logger.Infof("recreating Temporal schedules for stack %q", r.stack)

	connectors, err := r.listActiveConnectors(ctx)
	if err != nil {
		return fmt.Errorf("failed to list connectors: %w", err)
	}

	r.logger.Infof("found %d active connector(s)", len(connectors))

	hadFailures := false
	for _, connector := range connectors {
		if err := r.recreateConnectorSchedules(ctx, connector); err != nil {
			r.logger.Errorf("failed to recreate schedules for connector %s: %v", connector.ID.String(), err)
			hadFailures = true
			continue
		}
	}

	if hadFailures {
		return errors.New("one or more connectors failed while recreating schedules")
	}

	r.logger.Infof("done recreating schedules")
	return nil
}

func (r *RecreateSchedules) listActiveConnectors(ctx context.Context) ([]models.Connector, error) {
	var result []models.Connector
	q := storage.NewListConnectorsQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.ConnectorQuery{}).
			WithPageSize(100),
	)

	for {
		page, err := r.storage.ConnectorsList(ctx, q)
		if err != nil {
			return nil, err
		}

		for _, c := range page.Data {
			if !c.ScheduledForDeletion {
				result = append(result, c)
			}
		}

		if !page.HasMore {
			break
		}

		if err := bunpaginate.UnmarshalCursor(page.Next, &q); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (r *RecreateSchedules) recreateConnectorSchedules(ctx context.Context, connector models.Connector) error {
	r.logger.Infof("processing connector %s (%s)", connector.Name, connector.ID.String())

	taskTree, err := r.storage.ConnectorTasksTreeGet(ctx, connector.ID)
	if err != nil {
		return fmt.Errorf("failed to get task tree: %w", err)
	}
	if taskTree == nil {
		r.logger.Infof("  no task tree found, skipping")
		return nil
	}

	var config models.Config
	if err := json.Unmarshal(connector.Config, &config); err != nil {
		return fmt.Errorf("failed to unmarshal connector config: %w", err)
	}

	if config.PollingPeriod == 0 {
		config.PollingPeriod = 30 * time.Minute
	}

	taskQueue := engine.GetDefaultTaskQueue(r.stack)

	// Phase 1: Recreate root schedules from the task tree
	r.logger.Infof("  phase 1: recreating root schedules from task tree")
	if err := r.recreateRootSchedules(ctx, *taskTree, connector.ID, config, taskQueue); err != nil {
		return fmt.Errorf("phase 1 (root schedules): %w", err)
	}

	// Phase 2: Recreate sub-schedules from the schedules DB table + account data
	r.logger.Infof("  phase 2: recreating sub-schedules from DB")
	if err := r.recreateSubSchedules(ctx, connector.ID, *taskTree, config, taskQueue); err != nil {
		return fmt.Errorf("phase 2 (sub-schedules): %w", err)
	}

	return nil
}

func (r *RecreateSchedules) recreateRootSchedules(
	ctx context.Context,
	tasks models.ConnectorTasksTree,
	connectorID models.ConnectorID,
	config models.Config,
	taskQueue string,
) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, task := range tasks {
		if !task.Periodically {
			continue
		}

		workflowName, capability, request := r.buildScheduleParams(task, connectorID, nil, config)
		if workflowName == "" {
			r.logger.Errorf("  unknown task type %d, skipping", task.TaskType)
			continue
		}

		scheduleID := fmt.Sprintf("%s-%s-%s", r.stack, connectorID.String(), capability.String())
		nextTasks := task.NextTasks

		g.Go(func() error {
			err := r.createSchedule(ctx, scheduleID, workflowName, config.PollingPeriod, taskQueue, request, nextTasks)
			if err != nil {
				r.logger.Errorf("  failed to create schedule %s: %v", scheduleID, err)
				return err
			}
			r.logger.Infof("  schedule %s ensured (workflow=%s, interval=%s)", scheduleID, workflowName, config.PollingPeriod)
			return nil
		})
	}

	return g.Wait()
}

// recreateSubSchedules lists all schedules stored in the DB for a connector,
// identifies sub-schedules (those with a fromPayload.ID suffix), looks up the
// corresponding account to reconstruct the PSP payload, and recreates the
// Temporal schedule.
func (r *RecreateSchedules) recreateSubSchedules(
	ctx context.Context,
	connectorID models.ConnectorID,
	taskTree models.ConnectorTasksTree,
	config models.Config,
	taskQueue string,
) error {
	prefix := fmt.Sprintf("%s-%s-", r.stack, connectorID.String())

	// Cache account lookups to avoid repeated DB queries for the same reference
	accountCache := make(map[string]*models.Account)

	q := storage.NewListSchedulesQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.ScheduleQuery{}).
			WithPageSize(100).
			WithQueryBuilder(
				query.Match("connector_id", connectorID.String()),
			),
	)

	g, ctx := errgroup.WithContext(ctx)

	for {
		page, err := r.storage.SchedulesList(ctx, q)
		if err != nil {
			return fmt.Errorf("failed to list schedules: %w", err)
		}

		for _, schedule := range page.Data {
			capabilityStr, payloadID, ok := r.parseScheduleID(schedule.ID, prefix)
			if !ok {
				r.logger.Errorf("  could not parse schedule ID %s, skipping", schedule.ID)
				continue
			}

			// Skip root schedules — already handled in phase 1
			if payloadID == "" {
				continue
			}

			capInfo := r.resolveCapability(capabilityStr)
			if capInfo == nil {
				r.logger.Errorf("  unknown capability %q in schedule %s, skipping", capabilityStr, schedule.ID)
				continue
			}

			// Look up account (with cache)
			account, ok := accountCache[payloadID]
			if !ok {
				account, err = r.storage.AccountsGet(ctx, models.AccountID{
					Reference:   payloadID,
					ConnectorID: connectorID,
				})
				if err != nil {
					r.logger.Errorf("  account %q not found for schedule %s, skipping: %v", payloadID, schedule.ID, err)
					continue
				}
				accountCache[payloadID] = account
			}

			pspAccount := models.PSPAccount{
				Reference:    account.Reference,
				CreatedAt:    account.CreatedAt,
				Name:         account.Name,
				DefaultAsset: account.DefaultAsset,
				Metadata:     account.Metadata,
				Raw:          account.Raw,
			}

			payload, err := json.Marshal(pspAccount)
			if err != nil {
				r.logger.Errorf("  failed to marshal account %q for schedule %s: %v", payloadID, schedule.ID, err)
				continue
			}

			fromPayload := &workflow.FromPayload{
				ID:      payloadID,
				Payload: payload,
			}

			task := r.findTaskForCapability(taskTree, capInfo.taskType)
			var nextTasks []models.ConnectorTaskTree
			if task != nil {
				nextTasks = task.NextTasks
			}
			request := r.buildRequestForCapability(capInfo, connectorID, fromPayload, config)

			sid := schedule.ID
			wfName := capInfo.workflowName
			g.Go(func() error {
				err := r.createSchedule(ctx, sid, wfName, config.PollingPeriod, taskQueue, request, nextTasks)
				if err != nil {
					r.logger.Errorf("  failed to create sub-schedule %s: %v", sid, err)
					return err
				}
				r.logger.Infof("  sub-schedule %s ensured (workflow=%s, account=%s)", sid, wfName, payloadID)
				return nil
			})
		}

		if !page.HasMore {
			break
		}

		if err := bunpaginate.UnmarshalCursor(page.Next, &q); err != nil {
			return fmt.Errorf("failed to unmarshal schedules cursor: %w", err)
		}
	}

	return g.Wait()
}

// parseScheduleID extracts the capability string and optional fromPayload.ID
// from a schedule ID by stripping the known prefix ({stack}-{connectorID}-).
//
// Schedule IDs follow the format set by scheduleNextWorkflow in plugin_workflow.go:
//   - Root: {stack}-{connectorID}-{CAPABILITY}
//   - Sub:  {stack}-{connectorID}-{CAPABILITY}-{fromPayload.ID}
//
// The capability names use underscores (e.g., FETCH_ACCOUNTS) so the dash after
// the capability unambiguously separates it from the payload ID.
func (r *RecreateSchedules) parseScheduleID(scheduleID, prefix string) (string, string, bool) {
	remainder, found := strings.CutPrefix(scheduleID, prefix)
	if !found || remainder == "" {
		return "", "", false
	}

	// Try to match against known capabilities (longest first to avoid partial matches).
	for _, cap := range knownCapabilities {
		if remainder == cap.name {
			return cap.name, "", true
		}
		if payloadID, found := strings.CutPrefix(remainder, cap.name+"-"); found {
			return cap.name, payloadID, true
		}
	}

	return "", "", false
}

func (r *RecreateSchedules) resolveCapability(capabilityStr string) *struct {
	name         string
	capability   models.Capability
	taskType     models.TaskType
	workflowName string
} {
	for i := range knownCapabilities {
		if knownCapabilities[i].name == capabilityStr {
			return &knownCapabilities[i]
		}
	}
	return nil
}

// findTaskForCapability walks the task tree recursively to find the task node
// for a given capability's task type.
func (r *RecreateSchedules) findTaskForCapability(tasks models.ConnectorTasksTree, targetType models.TaskType) *models.ConnectorTaskTree {
	for i, task := range tasks {
		if task.TaskType == targetType {
			return &tasks[i]
		}
		if result := r.findTaskForCapability(task.NextTasks, targetType); result != nil {
			return result
		}
	}
	return nil
}

func (r *RecreateSchedules) buildRequestForCapability(
	capInfo *struct {
		name         string
		capability   models.Capability
		taskType     models.TaskType
		workflowName string
	},
	connectorID models.ConnectorID,
	fromPayload *workflow.FromPayload,
	config models.Config,
) any {
	name := strings.ToLower(capInfo.name)
	switch capInfo.taskType {
	case models.TASK_FETCH_ACCOUNTS:
		return workflow.FetchNextAccounts{Config: config, ConnectorID: connectorID, FromPayload: fromPayload, Periodically: true}
	case models.TASK_FETCH_BALANCES:
		return workflow.FetchNextBalances{Config: config, ConnectorID: connectorID, FromPayload: fromPayload, Periodically: true}
	case models.TASK_FETCH_EXTERNAL_ACCOUNTS:
		return workflow.FetchNextExternalAccounts{Config: config, ConnectorID: connectorID, FromPayload: fromPayload, Periodically: true}
	case models.TASK_FETCH_OTHERS:
		return workflow.FetchNextOthers{Config: config, ConnectorID: connectorID, Name: name, FromPayload: fromPayload, Periodically: true}
	case models.TASK_FETCH_PAYMENTS:
		return workflow.FetchNextPayments{Config: config, ConnectorID: connectorID, FromPayload: fromPayload, Periodically: true}
	case models.TASK_CREATE_WEBHOOKS:
		return workflow.CreateWebhooks{Config: config, ConnectorID: connectorID, FromPayload: fromPayload}
	default:
		return nil
	}
}

func (r *RecreateSchedules) buildScheduleParams(
	task models.ConnectorTaskTree,
	connectorID models.ConnectorID,
	fromPayload *workflow.FromPayload,
	config models.Config,
) (workflowName string, capability models.Capability, request any) {
	switch task.TaskType {
	case models.TASK_FETCH_ACCOUNTS:
		return workflow.RunFetchNextAccounts, models.CAPABILITY_FETCH_ACCOUNTS, workflow.FetchNextAccounts{
			Config: config, ConnectorID: connectorID, FromPayload: fromPayload, Periodically: true,
		}
	case models.TASK_FETCH_BALANCES:
		return workflow.RunFetchNextBalances, models.CAPABILITY_FETCH_BALANCES, workflow.FetchNextBalances{
			Config: config, ConnectorID: connectorID, FromPayload: fromPayload, Periodically: true,
		}
	case models.TASK_FETCH_EXTERNAL_ACCOUNTS:
		return workflow.RunFetchNextExternalAccounts, models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS, workflow.FetchNextExternalAccounts{
			Config: config, ConnectorID: connectorID, FromPayload: fromPayload, Periodically: true,
		}
	case models.TASK_FETCH_OTHERS:
		return workflow.RunFetchNextOthers, models.CAPABILITY_FETCH_OTHERS, workflow.FetchNextOthers{
			Config: config, ConnectorID: connectorID, Name: strings.ToLower(models.CAPABILITY_FETCH_OTHERS.String()), FromPayload: fromPayload, Periodically: true,
		}
	case models.TASK_FETCH_PAYMENTS:
		return workflow.RunFetchNextPayments, models.CAPABILITY_FETCH_PAYMENTS, workflow.FetchNextPayments{
			Config: config, ConnectorID: connectorID, FromPayload: fromPayload, Periodically: true,
		}
	case models.TASK_CREATE_WEBHOOKS:
		return workflow.RunCreateWebhooks, models.CAPABILITY_CREATE_WEBHOOKS, workflow.CreateWebhooks{
			Config: config, ConnectorID: connectorID, FromPayload: fromPayload,
		}
	default:
		return "", 0, nil
	}
}

func (r *RecreateSchedules) createSchedule(
	ctx context.Context,
	scheduleID string,
	workflowName string,
	pollingPeriod time.Duration,
	taskQueue string,
	request any,
	nextTasks []models.ConnectorTaskTree,
) error {
	jitter := pollingPeriod / 2
	maxJitter := 5 * time.Minute
	if jitter > maxJitter {
		jitter = maxJitter
	}

	_, err := r.temporalClient.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: scheduleID,
		Spec: client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{
				{Every: pollingPeriod},
			},
			Jitter: jitter,
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        scheduleID,
			Workflow:  workflowName,
			Args:      []any{request, nextTasks},
			TaskQueue: taskQueue,
			TypedSearchAttributes: temporal.NewSearchAttributes(
				temporal.NewSearchAttributeKeyKeyword(workflow.SearchAttributeScheduleID).ValueSet(scheduleID),
				temporal.NewSearchAttributeKeyKeyword(workflow.SearchAttributeStack).ValueSet(r.stack),
			),
		},
		Overlap:            enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
		TriggerImmediately: true,
		SearchAttributes: map[string]any{
			workflow.SearchAttributeScheduleID: scheduleID,
			workflow.SearchAttributeStack:      r.stack,
		},
	})

	if err != nil {
		var already *serviceerror.AlreadyExists
		var wfAlreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &wfAlreadyStarted) || errors.As(err, &already) {
			return nil
		}
		if errors.Is(err, temporal.ErrScheduleAlreadyRunning) {
			return nil
		}
		return err
	}

	return nil
}
