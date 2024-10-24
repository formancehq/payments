package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/contextutil"
	"github.com/formancehq/payments/internal/connectors/engine/plugins"
	"github.com/formancehq/payments/internal/connectors/engine/webhooks"
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	"github.com/formancehq/payments/internal/models"
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
	ForwardBankAccount(ctx context.Context, bankAccountID uuid.UUID, connectorID models.ConnectorID) (*models.BankAccount, error)
	// Create a transfer between two accounts on the given connector (PSP).
	CreateTransfer(ctx context.Context, piID models.PaymentInitiationID, attempt int) error
	// Create a payout on the given connector (PSP).
	CreatePayout(ctx context.Context, piID models.PaymentInitiationID, attempt int) error

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
	config := models.DefaultConfig()
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return models.ConnectorID{}, err
	}

	if err := config.Validate(); err != nil {
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
	ctx, _ = contextutil.Detached(ctx)
	// Since we detached the context, we need to wait for the operation to finish
	// even if the app is shutting down gracefully.
	e.wg.Add(1)
	defer e.wg.Done()

	if err := e.storage.ConnectorsInstall(ctx, connector); err != nil {
		return models.ConnectorID{}, err
	}

	err := e.plugins.RegisterPlugin(connector.ID)
	if err != nil {
		return models.ConnectorID{}, handlePluginError(err)
	}

	err = e.workers.AddWorker(connector.ID.String())
	if err != nil {
		return models.ConnectorID{}, err
	}

	// Launch the workflow
	run, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("install-%s", connector.ID.String()),
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
		return models.ConnectorID{}, err
	}

	// Wait for installation to complete
	if err := run.Get(ctx, nil); err != nil {
		return models.ConnectorID{}, err
	}

	return connector.ID, nil
}

func (e *engine) UninstallConnector(ctx context.Context, connectorID models.ConnectorID) error {
	if err := e.workers.RemoveWorker(connectorID.String()); err != nil {
		return err
	}

	if err := e.storage.ConnectorsScheduleForDeletion(ctx, connectorID); err != nil {
		return err
	}

	// Launch the uninstallation in background
	_, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("uninstall-%s", connectorID.String()),
			TaskQueue:                                defaultWorkerName,
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunUninstallConnector,
		workflow.UninstallConnector{
			ConnectorID:       connectorID,
			DefaultWorkerName: defaultWorkerName,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (e *engine) ResetConnector(ctx context.Context, connectorID models.ConnectorID) error {
	connector, err := e.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		return err
	}

	// Detached the context to avoid being in a weird state if request is
	// cancelled in the middle of the operation.
	ctx, _ = contextutil.Detached(ctx)
	// Since we detached the context, we need to wait for the operation to finish
	// even if the app is shutting down gracefully.
	e.wg.Add(1)
	defer e.wg.Done()

	if err := e.UninstallConnector(ctx, connectorID); err != nil {
		return err
	}

	_, err = e.InstallConnector(ctx, connectorID.Provider, connector.Config)
	if err != nil {
		return err
	}

	run, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                    fmt.Sprintf("reset-%s", connectorID.String()),
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
		return err
	}

	if err := run.Get(ctx, nil); err != nil {
		return err
	}

	return nil
}

func (e *engine) CreateFormanceAccount(ctx context.Context, account models.Account) error {
	if err := e.storage.AccountsUpsert(ctx, []models.Account{account}); err != nil {
		return err
	}

	// Do not wait for sending of events
	_, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("create-formance-account-send-events-%s-%s", account.ConnectorID.String(), account.Reference),
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
		return err
	}

	return nil
}

func (e *engine) CreateFormancePayment(ctx context.Context, payment models.Payment) error {
	if err := e.storage.PaymentsUpsert(ctx, []models.Payment{payment}); err != nil {
		return err
	}

	// Do not wait for sending of events
	_, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("create-formance-payment-send-events-%s-%s", payment.ConnectorID.String(), payment.Reference),
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
		return err
	}

	return nil
}

func (e *engine) ForwardBankAccount(ctx context.Context, bankAccountID uuid.UUID, connectorID models.ConnectorID) (*models.BankAccount, error) {
	run, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("create-bank-account-%s-%s", connectorID.String(), bankAccountID.String()),
			TaskQueue:                                connectorID.String(),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunCreateBankAccount,
		workflow.CreateBankAccount{
			ConnectorID:   connectorID,
			BankAccountID: bankAccountID,
		},
	)
	if err != nil {
		return nil, err
	}

	var bankAccount models.BankAccount
	// Wait for bank account creation to complete
	if err := run.Get(ctx, &bankAccount); err != nil {
		return nil, err
	}

	return &bankAccount, nil
}

func (e *engine) CreateTransfer(ctx context.Context, piID models.PaymentInitiationID, attempt int) error {
	run, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("create-transfer-%s-%s-%d", piID.ConnectorID.String(), piID.String(), attempt),
			TaskQueue:                                piID.ConnectorID.String(),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunCreateTransfer,
		workflow.CreateTransfer{
			ConnectorID:         piID.ConnectorID,
			PaymentInitiationID: piID,
		},
	)
	if err != nil {
		return err
	}

	// Wait for bank account creation to complete
	if err := run.Get(ctx, nil); err != nil {
		return err
	}

	return nil
}

func (e *engine) CreatePayout(ctx context.Context, piID models.PaymentInitiationID, attempt int) error {
	run, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("create-payout-%s-%s-%d", piID.ConnectorID.String(), piID.String(), attempt),
			TaskQueue:                                piID.ConnectorID.String(),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunCreatePayout,
		workflow.CreatePayout{
			ConnectorID:         piID.ConnectorID,
			PaymentInitiationID: piID,
		},
	)
	if err != nil {
		return err
	}

	// Wait for bank account creation to complete
	if err := run.Get(ctx, nil); err != nil {
		return err
	}

	return nil
}

func (e *engine) HandleWebhook(ctx context.Context, urlPath string, webhook models.Webhook) error {
	configs, err := e.webhooks.GetConfigs(webhook.ConnectorID, urlPath)
	if err != nil {
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
		return errors.New("webhook config not found")
	}

	ctx, _ = contextutil.Detached(ctx)
	// Since we detached the context, we need to wait for the operation to finish
	// even if the app is shutting down gracefully.
	e.wg.Add(1)
	defer e.wg.Done()

	if _, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("webhook-%s-%s", webhook.ConnectorID.String(), webhook.ID),
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
		return err
	}

	return nil
}

func (e *engine) CreatePool(ctx context.Context, pool models.Pool) error {
	if err := e.storage.PoolsUpsert(ctx, pool); err != nil {
		return err
	}

	// Do not wait for sending of events
	_, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("pools-creation-%s", pool.IdempotencyKey()),
			TaskQueue:                                defaultWorkerName,
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
		return err
	}

	return nil
}

func (e *engine) AddAccountToPool(ctx context.Context, id uuid.UUID, accountID models.AccountID) error {
	if err := e.storage.PoolsAddAccount(ctx, id, accountID); err != nil {
		return err
	}

	ctx, _ = contextutil.Detached(ctx)
	// Since we detached the context, we need to wait for the operation to finish
	// even if the app is shutting down gracefully.
	e.wg.Add(1)
	defer e.wg.Done()

	pool, err := e.storage.PoolsGet(ctx, id)
	if err != nil {
		return err
	}

	// Do not wait for sending of events
	_, err = e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("pools-add-account-%s", pool.IdempotencyKey()),
			TaskQueue:                                defaultWorkerName,
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
		return err
	}

	return nil
}

func (e *engine) RemoveAccountFromPool(ctx context.Context, id uuid.UUID, accountID models.AccountID) error {
	if err := e.storage.PoolsRemoveAccount(ctx, id, accountID); err != nil {
		return err
	}

	ctx, _ = contextutil.Detached(ctx)
	// Since we detached the context, we need to wait for the operation to finish
	// even if the app is shutting down gracefully.
	e.wg.Add(1)
	defer e.wg.Done()

	pool, err := e.storage.PoolsGet(ctx, id)
	if err != nil {
		return err
	}

	// Do not wait for sending of events
	_, err = e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("pools-remove-account-%s", pool.IdempotencyKey()),
			TaskQueue:                                defaultWorkerName,
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
		return err
	}

	return nil
}

func (e *engine) DeletePool(ctx context.Context, poolID uuid.UUID) error {
	if err := e.storage.PoolsDelete(ctx, poolID); err != nil {
		return err
	}

	// Do not wait for sending of events
	_, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("pools-deletion-%s", poolID.String()),
			TaskQueue:                                defaultWorkerName,
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
	err := e.plugins.RegisterPlugin(connector.ID)
	if err != nil {
		return err
	}

	if !connector.ScheduledForDeletion {
		err = e.workers.AddWorker(connector.ID.String())
		if err != nil {
			return err
		}

		config := models.DefaultConfig()
		if err := json.Unmarshal(connector.Config, &config); err != nil {
			return err
		}

		// Launch the workflow
		_, err = e.temporalClient.ExecuteWorkflow(
			ctx,
			client.StartWorkflowOptions{
				ID:                                       fmt.Sprintf("install-%s", connector.ID.String()),
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
