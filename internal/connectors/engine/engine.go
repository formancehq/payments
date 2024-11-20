package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/connectors/engine/plugins"
	"github.com/formancehq/payments/internal/connectors/engine/webhooks"
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

type Engine interface {
	// Install a connector with the given provider and configuration.
	InstallConnector(ctx context.Context, provider string, rawConfig json.RawMessage) (models.ConnectorID, error)
	// Uninstall a connector with the given ID.
	UninstallConnector(ctx context.Context, connectorID models.ConnectorID) error
	// Reset a connector with the given ID, by uninstalling and reinstalling it.
	ResetConnector(ctx context.Context, connectorID models.ConnectorID) error

	// Create a Formance account, no call to the plugin, just a creation
	// of an account in the database related to the provided connector id.
	CreateFormanceAccount(ctx context.Context, account models.Account) error
	// Create a Formance payment, no call to the plugin, just a creation
	// of a payment in the database related to the provided connector id.
	CreateFormancePayment(ctx context.Context, payment models.Payment) error

	// Forward a bank account to the given connector, which will create it
	// in the external system (PSP).
	ForwardBankAccount(ctx context.Context, bankAccountID uuid.UUID, connectorID models.ConnectorID, waitResult bool) (models.Task, error)
	// Create a transfer between two accounts on the given connector (PSP).
	CreateTransfer(ctx context.Context, piID models.PaymentInitiationID, attempt int, waitResult bool) (models.Task, error)
	// Create a payout on the given connector (PSP).
	CreatePayout(ctx context.Context, piID models.PaymentInitiationID, attempt int, waitResult bool) (models.Task, error)

	// We received a webhook, handle it by calling the corresponding plugin to
	// translate it to a formance object and store it.
	HandleWebhook(ctx context.Context, urlPath string, webhook models.Webhook) error

	// Create a Formance pool composed of accounts.
	CreatePool(ctx context.Context, pool models.Pool) error
	// Add an account to a Formance pool.
	AddAccountToPool(ctx context.Context, id uuid.UUID, accountID models.AccountID) error
	// Remove an account from a Formance pool.
	RemoveAccountFromPool(ctx context.Context, id uuid.UUID, accountID models.AccountID) error
	// Delete a Formance pool.
	DeletePool(ctx context.Context, poolID uuid.UUID) error

	// Called when the engine is starting, to start all the plugins.
	OnStart(ctx context.Context) error
	// Called when the engine is stopping, to stop all the plugins.
	OnStop(ctx context.Context)
}

type engine struct {
	temporalClient client.Client

	workers  *Workers
	plugins  plugins.Plugins
	storage  storage.Storage
	webhooks webhooks.Webhooks

	stack string

	wg sync.WaitGroup
}

func New(temporalClient client.Client, workers *Workers, plugins plugins.Plugins, storage storage.Storage, webhooks webhooks.Webhooks, stack string) Engine {
	return &engine{
		temporalClient: temporalClient,
		workers:        workers,
		plugins:        plugins,
		storage:        storage,
		webhooks:       webhooks,
		stack:          stack,
		wg:             sync.WaitGroup{},
	}
}

func (e *engine) InstallConnector(ctx context.Context, provider string, rawConfig json.RawMessage) (models.ConnectorID, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.InstallConnector")
	defer span.End()

	config := models.DefaultConfig()
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		otel.RecordError(span, err)
		return models.ConnectorID{}, err
	}

	if err := config.Validate(); err != nil {
		otel.RecordError(span, err)
		return models.ConnectorID{}, errors.Wrap(ErrValidation, err.Error())
	}

	connector := models.Connector{
		ID: models.ConnectorID{
			Reference: uuid.New(),
			Provider:  provider,
		},
		Name:      config.Name,
		CreatedAt: time.Now().UTC(),
		Provider:  provider,
		Config:    rawConfig,
	}

	// Detached the context to avoid being in a weird state if request is
	// cancelled in the middle of the operation.
	detachedCtx := context.WithoutCancel(ctx)
	// Since we detached the context, we need to wait for the operation to finish
	// even if the app is shutting down gracefully.
	e.wg.Add(1)
	defer e.wg.Done()

	if err := e.storage.ConnectorsInstall(detachedCtx, connector); err != nil {
		otel.RecordError(span, err)
		return models.ConnectorID{}, err
	}

	err := e.plugins.RegisterPlugin(connector.ID, config)
	if err != nil {
		otel.RecordError(span, err)
		return models.ConnectorID{}, handlePluginError(err)
	}

	err = e.workers.AddWorker(connector.ID.String())
	if err != nil {
		otel.RecordError(span, err)
		return models.ConnectorID{}, err
	}

	// Launch the workflow
	run, err := e.temporalClient.ExecuteWorkflow(
		detachedCtx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("install-%s-%s", e.stack, connector.ID.String()),
			TaskQueue:                                connector.ID.String(),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunInstallConnector,
		workflow.InstallConnector{
			ConnectorID: connector.ID,
			RawConfig:   rawConfig,
			Config:      config,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return models.ConnectorID{}, err
	}

	// Wait for installation to complete in order to return connector ID through API
	if err := run.Get(ctx, nil); err != nil {
		otel.RecordError(span, err)
		return models.ConnectorID{}, err
	}

	return connector.ID, nil
}

func (e *engine) UninstallConnector(ctx context.Context, connectorID models.ConnectorID) error {
	ctx, span := otel.Tracer().Start(ctx, "engine.UninstallConnector")
	defer span.End()

	if err := e.workers.RemoveWorker(connectorID.String()); err != nil {
		otel.RecordError(span, err)
		return err
	}

	if err := e.storage.ConnectorsScheduleForDeletion(ctx, connectorID); err != nil {
		otel.RecordError(span, err)
		return err
	}

	// Launch the uninstallation in background
	_, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("uninstall-%s-%s", e.stack, connectorID.String()),
			TaskQueue:                                e.workers.GetDefaultWorker(),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunUninstallConnector,
		workflow.UninstallConnector{
			ConnectorID:       connectorID,
			DefaultWorkerName: e.workers.GetDefaultWorker(),
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	return nil
}

func (e *engine) ResetConnector(ctx context.Context, connectorID models.ConnectorID) error {
	ctx, span := otel.Tracer().Start(ctx, "engine.ResetConnector")
	defer span.End()

	connector, err := e.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	// Detached the context to avoid being in a weird state if request is
	// cancelled in the middle of the operation.
	detachedCtx := context.WithoutCancel(ctx)
	// Since we detached the context, we need to wait for the operation to finish
	// even if the app is shutting down gracefully.
	e.wg.Add(1)
	defer e.wg.Done()

	if err := e.UninstallConnector(detachedCtx, connectorID); err != nil {
		otel.RecordError(span, err)
		return err
	}

	_, err = e.InstallConnector(detachedCtx, connectorID.Provider, connector.Config)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	run, err := e.temporalClient.ExecuteWorkflow(
		detachedCtx,
		client.StartWorkflowOptions{
			ID:                    fmt.Sprintf("reset-%s-%s", e.stack, connectorID.String()),
			TaskQueue:             connectorID.String(),
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunSendEvents,
		workflow.SendEvents{
			ConnectorReset: &connectorID,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	if err := run.Get(ctx, nil); err != nil {
		otel.RecordError(span, err)
		return err
	}
	return nil
}

func (e *engine) CreateFormanceAccount(ctx context.Context, account models.Account) error {
	ctx, span := otel.Tracer().Start(ctx, "engine.CreateFormanceAccount")
	defer span.End()

	capabilities, err := e.plugins.GetCapabilities(account.ConnectorID)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	_, ok := capabilities[models.CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION]
	if !ok {
		otel.RecordError(span, err)
		return errors.New("connector does not support account creation")
	}

	if err := e.storage.AccountsUpsert(ctx, []models.Account{account}); err != nil {
		otel.RecordError(span, err)
		return err
	}

	// Do not wait for sending of events
	_, err = e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("create-formance-account-send-events-%s-%s-%s", e.stack, account.ConnectorID.String(), account.Reference),
			TaskQueue:                                account.ConnectorID.String(),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunSendEvents,
		workflow.SendEvents{
			Account: &account,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	return nil
}

func (e *engine) CreateFormancePayment(ctx context.Context, payment models.Payment) error {
	ctx, span := otel.Tracer().Start(ctx, "engine.CreateFormancePayment")
	defer span.End()

	capabilities, err := e.plugins.GetCapabilities(payment.ConnectorID)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	_, ok := capabilities[models.CAPABILITY_ALLOW_FORMANCE_PAYMENT_CREATION]
	if !ok {
		otel.RecordError(span, err)
		return errors.New("connector does not support payment creation")
	}

	if err := e.storage.PaymentsUpsert(ctx, []models.Payment{payment}); err != nil {
		otel.RecordError(span, err)
		return err
	}

	// Do not wait for sending of events
	_, err = e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("create-formance-payment-send-events-%s-%s-%s", e.stack, payment.ConnectorID.String(), payment.Reference),
			TaskQueue:                                payment.ConnectorID.String(),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunSendEvents,
		workflow.SendEvents{
			Payment: &payment,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	return nil
}

func (e *engine) ForwardBankAccount(ctx context.Context, bankAccountID uuid.UUID, connectorID models.ConnectorID, waitResult bool) (models.Task, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.ForwardBankAccount")
	defer span.End()

	id := models.TaskIDReference(fmt.Sprintf("create-bank-account-%s", e.stack), connectorID, bankAccountID.String())

	now := time.Now().UTC()
	task := models.Task{
		ID: models.TaskID{
			Reference:   id,
			ConnectorID: connectorID,
		},
		ConnectorID: connectorID,
		Status:      models.TASK_STATUS_PROCESSING,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := e.storage.TasksUpsert(ctx, task); err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	run, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       id,
			TaskQueue:                                connectorID.String(),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunCreateBankAccount,
		workflow.CreateBankAccount{
			TaskID:        task.ID,
			ConnectorID:   connectorID,
			BankAccountID: bankAccountID,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	if waitResult {
		// Wait for bank account creation to complete
		if err := run.Get(ctx, nil); err != nil {
			otel.RecordError(span, err)
			return models.Task{}, err
		}
	}

	return task, nil
}

func (e *engine) CreateTransfer(ctx context.Context, piID models.PaymentInitiationID, attempt int, waitResult bool) (models.Task, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.CreateTransfer")
	defer span.End()

	id := models.TaskIDReference(fmt.Sprintf("create-transfer-%s-%d", e.stack, attempt), piID.ConnectorID, piID.String())

	now := time.Now().UTC()
	task := models.Task{
		ID: models.TaskID{
			Reference:   id,
			ConnectorID: piID.ConnectorID,
		},
		ConnectorID: piID.ConnectorID,
		Status:      models.TASK_STATUS_PROCESSING,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := e.storage.TasksUpsert(ctx, task); err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	run, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       id,
			TaskQueue:                                piID.ConnectorID.String(),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunCreateTransfer,
		workflow.CreateTransfer{
			TaskID:              task.ID,
			ConnectorID:         piID.ConnectorID,
			PaymentInitiationID: piID,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	if waitResult {
		// Wait for bank account creation to complete
		if err := run.Get(ctx, nil); err != nil {
			otel.RecordError(span, err)
			return models.Task{}, err
		}
	}

	return task, nil
}

func (e *engine) CreatePayout(ctx context.Context, piID models.PaymentInitiationID, attempt int, waitResult bool) (models.Task, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.CreatePayout")
	defer span.End()

	id := models.TaskIDReference(fmt.Sprintf("create-payout-%s-%d", e.stack, attempt), piID.ConnectorID, piID.String())

	now := time.Now().UTC()
	task := models.Task{
		ID: models.TaskID{
			Reference:   id,
			ConnectorID: piID.ConnectorID,
		},
		ConnectorID: piID.ConnectorID,
		Status:      models.TASK_STATUS_PROCESSING,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := e.storage.TasksUpsert(ctx, task); err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	run, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       id,
			TaskQueue:                                piID.ConnectorID.String(),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunCreatePayout,
		workflow.CreatePayout{
			TaskID:              task.ID,
			ConnectorID:         piID.ConnectorID,
			PaymentInitiationID: piID,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	if waitResult {
		// Wait for bank account creation to complete
		if err := run.Get(ctx, nil); err != nil {
			otel.RecordError(span, err)
			return models.Task{}, err
		}
	}

	return task, nil
}

func (e *engine) HandleWebhook(ctx context.Context, urlPath string, webhook models.Webhook) error {
	ctx, span := otel.Tracer().Start(ctx, "engine.HandleWebhook")
	defer span.End()

	configs, err := e.webhooks.GetConfigs(webhook.ConnectorID, urlPath)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	var config *models.WebhookConfig
	for _, c := range configs {
		if !strings.Contains(urlPath, c.URLPath) {
			continue
		}

		config = &c
		break
	}

	if config == nil {
		err := errors.New("webhook config not found")
		otel.RecordError(span, err)
		return err
	}

	detachedCtx := context.WithoutCancel(ctx)
	// Since we detached the context, we need to wait for the operation to finish
	// even if the app is shutting down gracefully.
	e.wg.Add(1)
	defer e.wg.Done()

	if _, err := e.temporalClient.ExecuteWorkflow(
		detachedCtx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("webhook-%s-%s-%s", e.stack, webhook.ConnectorID.String(), webhook.ID),
			TaskQueue:                                webhook.ConnectorID.String(),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunHandleWebhooks,
		workflow.HandleWebhooks{
			ConnectorID:   webhook.ConnectorID,
			WebhookConfig: *config,
			Webhook:       webhook,
		},
	); err != nil {
		otel.RecordError(span, err)
		return err
	}

	return nil
}

func (e *engine) CreatePool(ctx context.Context, pool models.Pool) error {
	ctx, span := otel.Tracer().Start(ctx, "engine.CreatePool")
	defer span.End()

	if err := e.storage.PoolsUpsert(ctx, pool); err != nil {
		otel.RecordError(span, err)
		return err
	}

	// Do not wait for sending of events
	_, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("pools-creation-%s-%s", e.stack, pool.IdempotencyKey()),
			TaskQueue:                                e.workers.GetDefaultWorker(),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunSendEvents,
		workflow.SendEvents{
			PoolsCreation: &pool,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	return nil
}

func (e *engine) AddAccountToPool(ctx context.Context, id uuid.UUID, accountID models.AccountID) error {
	ctx, span := otel.Tracer().Start(ctx, "engine.AddAccountToPool")
	defer span.End()

	if err := e.storage.PoolsAddAccount(ctx, id, accountID); err != nil {
		otel.RecordError(span, err)
		return err
	}

	detachedCtx := context.WithoutCancel(ctx)
	// Since we detached the context, we need to wait for the operation to finish
	// even if the app is shutting down gracefully.
	e.wg.Add(1)
	defer e.wg.Done()

	pool, err := e.storage.PoolsGet(detachedCtx, id)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	// Do not wait for sending of events
	_, err = e.temporalClient.ExecuteWorkflow(
		detachedCtx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("pools-add-account-%s-%s", e.stack, pool.IdempotencyKey()),
			TaskQueue:                                e.workers.GetDefaultWorker(),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunSendEvents,
		workflow.SendEvents{
			PoolsCreation: pool,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	return nil
}

func (e *engine) RemoveAccountFromPool(ctx context.Context, id uuid.UUID, accountID models.AccountID) error {
	ctx, span := otel.Tracer().Start(ctx, "engine.RemoveAccountFromPool")
	defer span.End()

	if err := e.storage.PoolsRemoveAccount(ctx, id, accountID); err != nil {
		otel.RecordError(span, err)
		return err
	}

	detachedCtx := context.WithoutCancel(ctx)
	// Since we detached the context, we need to wait for the operation to finish
	// even if the app is shutting down gracefully.
	e.wg.Add(1)
	defer e.wg.Done()

	pool, err := e.storage.PoolsGet(detachedCtx, id)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	// Do not wait for sending of events
	_, err = e.temporalClient.ExecuteWorkflow(
		detachedCtx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("pools-remove-account-%s-%s", e.stack, pool.IdempotencyKey()),
			TaskQueue:                                e.workers.GetDefaultWorker(),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunSendEvents,
		workflow.SendEvents{
			PoolsCreation: pool,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	return nil
}

func (e *engine) DeletePool(ctx context.Context, poolID uuid.UUID) error {
	ctx, span := otel.Tracer().Start(ctx, "engine.DeletePool")
	defer span.End()

	if err := e.storage.PoolsDelete(ctx, poolID); err != nil {
		otel.RecordError(span, err)
		return err
	}

	// Do not wait for sending of events
	_, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("pools-deletion-%s-%s", e.stack, poolID.String()),
			TaskQueue:                                e.workers.GetDefaultWorker(),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunSendEvents,
		workflow.SendEvents{
			PoolsDeletion: &poolID,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	return nil
}

func (e *engine) OnStop(ctx context.Context) {
	waitingChan := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(waitingChan)
	}()

	select {
	case <-waitingChan:
	case <-ctx.Done():
	}
}

func (e *engine) OnStart(ctx context.Context) error {
	query := storage.NewListConnectorsQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.ConnectorQuery{}).
			WithPageSize(100),
	)

	for {
		connectors, err := e.storage.ConnectorsList(ctx, query)
		if err != nil {
			return err
		}

		for _, connector := range connectors.Data {
			if err := e.onStartPlugin(ctx, connector); err != nil {
				return err
			}
		}

		if !connectors.HasMore {
			break
		}

		err = bunpaginate.UnmarshalCursor(connectors.Next, &query)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *engine) onStartPlugin(ctx context.Context, connector models.Connector) error {
	// Even if the connector is scheduled for deletion, we still need to register
	// the plugin to be able to handle the uninstallation.
	// It will be unregistered when the uninstallation is done in the workflow
	// after the deletion of the connector entry in the database.
	config := models.DefaultConfig()
	if err := json.Unmarshal(connector.Config, &config); err != nil {
		return err
	}

	err := e.plugins.RegisterPlugin(connector.ID, config)
	if err != nil {
		return err
	}

	if !connector.ScheduledForDeletion {
		err = e.workers.AddWorker(connector.ID.String())
		if err != nil {
			return err
		}

		// Launch the workflow
		_, err = e.temporalClient.ExecuteWorkflow(
			ctx,
			client.StartWorkflowOptions{
				ID:                                       fmt.Sprintf("install-%s-%s", e.stack, connector.ID.String()),
				TaskQueue:                                connector.ID.String(),
				WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
				WorkflowExecutionErrorWhenAlreadyStarted: false,
				SearchAttributes: map[string]interface{}{
					workflow.SearchAttributeStack: e.stack,
				},
			},
			workflow.RunInstallConnector,
			workflow.InstallConnector{
				ConnectorID: connector.ID,
				RawConfig:   connector.Config,
				Config:      config,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

var _ Engine = &engine{}
