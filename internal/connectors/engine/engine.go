package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/engine/plugins"
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/formancehq/payments/internal/storage"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"golang.org/x/sync/errgroup"
)

//go:generate mockgen -source engine.go -destination engine_generated.go -package engine . Engine
type Engine interface {
	// Install a connector with the given provider and configuration.
	InstallConnector(ctx context.Context, provider string, rawConfig json.RawMessage) (models.ConnectorID, error)
	// Uninstall a connector with the given ID.
	UninstallConnector(ctx context.Context, connectorID models.ConnectorID) (models.Task, error)
	// Reset a connector with the given ID, by uninstalling and reinstalling it.
	ResetConnector(ctx context.Context, connectorID models.ConnectorID) (models.Task, error)
	// Update a connector with the given configuration.
	UpdateConnector(ctx context.Context, connectorID models.ConnectorID, rawConfig json.RawMessage) error

	// Create a Formance account, no call to the plugin, just a creation
	// of an account in the database related to the provided connector id.
	CreateFormanceAccount(ctx context.Context, account models.Account) error
	// Create a Formance payment, no call to the plugin, just a creation
	// of a payment in the database related to the provided connector id.
	CreateFormancePayment(ctx context.Context, payment models.Payment) error
	// Create a Formance payment initiation, no call to the plugin, just a creation
	// of a payment initiation in the database and the sending of the related event.
	CreateFormancePaymentInitiation(ctx context.Context, paymentInitiation models.PaymentInitiation, adj models.PaymentInitiationAdjustment) error

	// Forward a bank account to the given connector, which will create it
	// in the external system (PSP).
	ForwardBankAccount(ctx context.Context, ba models.BankAccount, connectorID models.ConnectorID, waitResult bool) (models.Task, error)
	// Create a transfer between two accounts on the given connector (PSP).
	CreateTransfer(ctx context.Context, piID models.PaymentInitiationID, attempt int, waitResult bool) (models.Task, error)
	// Reverse a transfer on the given connector (PSP).
	ReverseTransfer(ctx context.Context, reversal models.PaymentInitiationReversal, waitResult bool) (models.Task, error)
	// Create a payout on the given connector (PSP).
	CreatePayout(ctx context.Context, piID models.PaymentInitiationID, attempt int, waitResult bool) (models.Task, error)
	// Reverse a payout on the given connector (PSP).
	ReversePayout(ctx context.Context, reversal models.PaymentInitiationReversal, waitResult bool) (models.Task, error)

	// Create a user on the given connector (PSP).
	ForwardUser(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) (models.Task, error)
	// Delete a user on the given connector (PSP).
	DeleteUser(ctx context.Context, psuID uuid.UUID) (models.Task, error)
	// Delete a user connection on the given connector (PSP).
	DeleteUserConnection(ctx context.Context, connectorID models.ConnectorID, psuID uuid.UUID, connectionID string) (models.Task, error)
	// Create a user link on the given connector (PSP).
	CreateUserLink(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, idempotencyKey *uuid.UUID, ClientRedirectURL *string) (models.Task, error)
	// Update a user link on the given connector (PSP).
	UpdateUserLink(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string, idempotencyKey *uuid.UUID, ClientRedirectURL *string) (models.Task, error)
	// Complete a user link on the given connector (PSP).
	CompleteUserLink(ctx context.Context, connectorID models.ConnectorID, attemptID uuid.UUID, httpCallInformation models.HTTPCallInformation) error

	// We received a webhook, handle it by calling the corresponding plugin to
	// translate it to a formance object and store it.
	HandleWebhook(ctx context.Context, url string, urlPath string, webhook models.Webhook) error

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
	logger logging.Logger

	temporalClient client.Client
	storage        storage.Storage

	// plugins is only really present in engine to allow validation of plugin configs prior to insert into the DB
	// other plugin-side work should be performed inside workers and not directly in the engine since we don't
	// have a listener function that checks for plugin updates or installs
	plugins plugins.Plugins

	stack          string
	stackPublicURL string

	wg sync.WaitGroup
}

func New(
	logger logging.Logger,
	temporalClient client.Client,
	storage storage.Storage,
	plugins plugins.Plugins,
	stack string,
	stackPublicURL string,
) Engine {
	return &engine{
		logger:         logger,
		temporalClient: temporalClient,
		storage:        storage,
		plugins:        plugins,
		stack:          stack,
		wg:             sync.WaitGroup{},
		stackPublicURL: stackPublicURL,
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
		return models.ConnectorID{}, errorsutils.NewWrappedError(err, ErrValidation)
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

	err := e.plugins.LoadPlugin(connector.ID, connector.Provider, connector.Name, config, connector.Config, false)
	if err != nil {
		otel.RecordError(span, err)
		if _, ok := err.(validator.ValidationErrors); ok || errors.Is(err, models.ErrInvalidConfig) {
			return models.ConnectorID{}, errorsutils.NewWrappedError(err, ErrValidation)
		}
		return models.ConnectorID{}, err
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

	// Launch the workflow
	run, err := e.launchInstallWorkflow(detachedCtx, connector, config)
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

func (e *engine) UninstallConnector(ctx context.Context, connectorID models.ConnectorID) (models.Task, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.UninstallConnector")
	defer span.End()

	detachedCtx := context.WithoutCancel(ctx)
	// Since we detached the context, we need to wait for the operation to finish
	// even if the app is shutting down gracefully.
	e.wg.Add(1)
	defer e.wg.Done()

	if err := e.storage.ConnectorsScheduleForDeletion(detachedCtx, connectorID); err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	now := time.Now()
	id := e.taskIDReferenceFor(IDPrefixConnectorUninstall, connectorID, "")
	task := models.Task{
		// Do not fill the connector ID as it will be deleted
		ID: models.TaskID{
			Reference:   id,
			ConnectorID: connectorID,
		},
		Status:    models.TASK_STATUS_PROCESSING,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := e.storage.TasksUpsert(ctx, task); err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	// Launch the uninstallation in background
	_, err := e.temporalClient.ExecuteWorkflow(
		detachedCtx,
		client.StartWorkflowOptions{
			ID:                                       id,
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunUninstallConnector,
		workflow.UninstallConnector{
			ConnectorID:       connectorID,
			DefaultWorkerName: GetDefaultTaskQueue(e.stack),
			TaskID:            &task.ID,
		},
	)
	if err != nil {
		task.Status = models.TASK_STATUS_FAILED
		task.UpdatedAt = time.Now()
		if err := e.storage.TasksUpsert(ctx, task); err != nil {
			e.logger.Errorf("failed to update task status to failed: %v", err)
		}

		otel.RecordError(span, err)
		return models.Task{}, err
	}

	return task, nil
}

func (e *engine) ResetConnector(ctx context.Context, connectorID models.ConnectorID) (models.Task, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.ResetConnector")
	defer span.End()

	detachedCtx := context.WithoutCancel(ctx)
	// Since we detached the context, we need to wait for the operation to finish
	// even if the app is shutting down gracefully.
	e.wg.Add(1)
	defer e.wg.Done()

	now := time.Now()
	id := e.taskIDReferenceFor(IDPrefixConnectorReset, connectorID, "")
	task := models.Task{
		// Do not fill the connector ID as it will be deleted
		ID: models.TaskID{
			Reference:   id,
			ConnectorID: connectorID,
		},
		Status:    models.TASK_STATUS_PROCESSING,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := e.storage.TasksUpsert(ctx, task); err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	_, err := e.temporalClient.ExecuteWorkflow(
		detachedCtx,
		client.StartWorkflowOptions{
			ID:                                       id,
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunResetConnector,
		workflow.ResetConnector{
			ConnectorID:       connectorID,
			DefaultWorkerName: GetDefaultTaskQueue(e.stack),
			TaskID:            task.ID,
		},
	)
	if err != nil {
		task.Status = models.TASK_STATUS_FAILED
		task.UpdatedAt = time.Now()
		if err := e.storage.TasksUpsert(ctx, task); err != nil {
			e.logger.Errorf("failed to update task status to failed: %v", err)
		}

		otel.RecordError(span, err)
		return models.Task{}, err
	}

	return task, nil
}

func (e *engine) UpdateConnector(ctx context.Context, connectorID models.ConnectorID, rawConfig json.RawMessage) error {
	ctx, span := otel.Tracer().Start(ctx, "engine.UpdateConnector")
	defer span.End()

	config := models.DefaultConfig()
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		otel.RecordError(span, err)
		return err
	}

	if err := config.Validate(); err != nil {
		otel.RecordError(span, err)
		return errors.Wrap(ErrValidation, err.Error())
	}

	connector, err := e.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		otel.RecordError(span, err)
		if errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("connector %w", ErrNotFound)
		}
	}

	if connector == nil {
		err := fmt.Errorf("connector %w", ErrNotFound)
		otel.RecordError(span, err)
		return err
	}

	connector.Config = rawConfig
	connector.Name = config.Name

	err = e.plugins.LoadPlugin(connector.ID, connector.Provider, connector.Name, config, connector.Config, true)
	if err != nil {
		otel.RecordError(span, err)
		if _, ok := err.(validator.ValidationErrors); ok || errors.Is(err, models.ErrInvalidConfig) {
			return errorsutils.NewWrappedError(err, ErrValidation)
		}
		return err
	}

	if err := e.storage.ConnectorsConfigUpdate(ctx, *connector); err != nil {
		otel.RecordError(span, err)
		return err
	}
	return nil
}

func (e *engine) CreateFormanceAccount(ctx context.Context, account models.Account) error {
	ctx, span := otel.Tracer().Start(ctx, "engine.CreateFormanceAccount")
	defer span.End()

	provider := models.ToV3Provider(account.ConnectorID.Provider)
	capabilities, err := registry.GetCapabilities(provider)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	found := false
	for _, c := range capabilities {
		if c == models.CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION {
			found = true
			break
		}
	}

	if !found {
		err := &ErrConnectorCapabilityNotSupported{Capability: "CreateFormanceAccount", Provider: provider}
		otel.RecordError(span, err)
		return err
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
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
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

	provider := models.ToV3Provider(payment.ConnectorID.Provider)
	capabilities, err := registry.GetCapabilities(provider)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	found := false
	for _, c := range capabilities {
		if c == models.CAPABILITY_ALLOW_FORMANCE_PAYMENT_CREATION {
			found = true
			break
		}
	}

	if !found {
		err := &ErrConnectorCapabilityNotSupported{Capability: "CreateFormancePayment", Provider: provider}
		otel.RecordError(span, err)
		return err
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
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
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

func (e *engine) CreateFormancePaymentInitiation(ctx context.Context, pi models.PaymentInitiation, adj models.PaymentInitiationAdjustment) error {
	ctx, span := otel.Tracer().Start(ctx, "engine.CreateFormancePaymentInitiation")
	defer span.End()

	if err := e.storage.PaymentInitiationsInsert(ctx, pi, adj); err != nil {
		otel.RecordError(span, err)
		return err
	}

	// Do not wait for sending of events
	_, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("create-payment-initiation-send-events-%s-%s-%s", e.stack, pi.ConnectorID.String(), pi.Reference),
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunSendEvents,
		workflow.SendEvents{
			PaymentInitiation:           &pi,
			PaymentInitiationAdjustment: &adj,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	return nil
}

func (e *engine) ForwardBankAccount(ctx context.Context, ba models.BankAccount, connectorID models.ConnectorID, waitResult bool) (models.Task, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.ForwardBankAccount")
	defer span.End()

	if _, err := e.storage.ConnectorsGet(ctx, connectorID); err != nil {
		otel.RecordError(span, err)
		if errors.Is(err, storage.ErrNotFound) {
			return models.Task{}, fmt.Errorf("connector %w", ErrNotFound)
		}
		return models.Task{}, err
	}

	id := e.taskIDReferenceFor(IDPrefixBankAccountCreate, connectorID, ba.ID.String())
	now := time.Now().UTC()
	task := models.Task{
		ID: models.TaskID{
			Reference:   id,
			ConnectorID: connectorID,
		},
		ConnectorID: &connectorID,
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
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunCreateBankAccount,
		workflow.CreateBankAccount{
			TaskID:      task.ID,
			ConnectorID: connectorID,
			BankAccount: ba,
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
		ConnectorID: &piID.ConnectorID,
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
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
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
			return models.Task{}, handleWorkflowError(err)
		}
	}

	return task, nil
}

func (e *engine) ReverseTransfer(ctx context.Context, reversal models.PaymentInitiationReversal, waitResult bool) (models.Task, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.ReverseTransfer")
	defer span.End()

	detachedCtx := context.WithoutCancel(ctx)
	e.wg.Add(1)
	defer e.wg.Done()

	id := models.TaskIDReference(fmt.Sprintf("reverse-transfer-%s-%s", e.stack, reversal.CreatedAt.String()), reversal.ConnectorID, reversal.ID.String())
	now := time.Now().UTC()
	task := models.Task{
		ID: models.TaskID{
			Reference:   id,
			ConnectorID: reversal.ConnectorID,
		},
		ConnectorID: &reversal.ConnectorID,
		Status:      models.TASK_STATUS_PROCESSING,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := e.storage.TasksUpsert(detachedCtx, task); err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	run, err := e.temporalClient.ExecuteWorkflow(
		detachedCtx,
		client.StartWorkflowOptions{
			ID:                                       id,
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunReverseTransfer,
		workflow.ReverseTransfer{
			TaskID:                      task.ID,
			ConnectorID:                 reversal.ConnectorID,
			PaymentInitiationReversalID: reversal.ID,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	if waitResult {
		// Wait for bank account creation to complete
		// use ctx instead of detachedCtx to allow the caller to cancel the operation
		// and not wait for the result
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
		ConnectorID: &piID.ConnectorID,
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
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
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
			return models.Task{}, handleWorkflowError(err)
		}
	}

	return task, nil
}

func (e *engine) ReversePayout(ctx context.Context, reversal models.PaymentInitiationReversal, waitResult bool) (models.Task, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.ReversePayout")
	defer span.End()

	detachedCtx := context.WithoutCancel(ctx)
	e.wg.Add(1)
	defer e.wg.Done()

	id := models.TaskIDReference(fmt.Sprintf("reverse-payout-%s-%s", e.stack, reversal.CreatedAt.String()), reversal.ConnectorID, reversal.ID.String())
	now := time.Now().UTC()
	task := models.Task{
		ID: models.TaskID{
			Reference:   id,
			ConnectorID: reversal.ConnectorID,
		},
		ConnectorID: &reversal.ConnectorID,
		Status:      models.TASK_STATUS_PROCESSING,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := e.storage.TasksUpsert(detachedCtx, task); err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	run, err := e.temporalClient.ExecuteWorkflow(
		detachedCtx,
		client.StartWorkflowOptions{
			ID:                                       id,
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunReversePayout,
		workflow.ReversePayout{
			TaskID:                      task.ID,
			ConnectorID:                 reversal.ConnectorID,
			PaymentInitiationReversalID: reversal.ID,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	if waitResult {
		// Wait for bank account creation to complete
		// use ctx instead of detachedCtx to allow the caller to cancel the operation
		// and not wait for the result
		if err := run.Get(ctx, nil); err != nil {
			otel.RecordError(span, err)
			return models.Task{}, err
		}
	}

	return task, nil
}

func (e *engine) ForwardUser(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) (models.Task, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.CreateUser")
	defer span.End()

	id := models.TaskIDReference(fmt.Sprintf("create-user-%s", e.stack), connectorID, psuID.String())
	now := time.Now().UTC()
	task := models.Task{
		ID: models.TaskID{
			Reference:   id,
			ConnectorID: connectorID,
		},
		ConnectorID: &connectorID,
		Status:      models.TASK_STATUS_PROCESSING,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := e.storage.TasksUpsert(ctx, task); err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	_, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       id,
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunCreateUser,
		workflow.CreateUser{
			TaskID:      task.ID,
			ConnectorID: connectorID,
			PsuID:       psuID,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	return task, nil
}

func (e *engine) DeleteUser(ctx context.Context, psuID uuid.UUID) (models.Task, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.DeleteUser")
	defer span.End()

	id := fmt.Sprintf("delete-user-%s-%s", e.stack, psuID.String())
	now := time.Now().UTC()
	task := models.Task{
		ID: models.TaskID{
			Reference: id,
		},
		Status:    models.TASK_STATUS_PROCESSING,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := e.storage.TasksUpsert(ctx, task); err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	_, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       id,
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunDeleteUser,
		workflow.DeleteUser{
			TaskID: task.ID,
			PsuID:  psuID,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	return task, nil
}

func (e *engine) DeleteUserConnection(ctx context.Context, connectorID models.ConnectorID, psuID uuid.UUID, connectionID string) (models.Task, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.DeleteUserConnection")
	defer span.End()

	id := models.TaskIDReference(fmt.Sprintf("delete-user-connection-%s", e.stack), connectorID, psuID.String())
	now := time.Now().UTC()
	task := models.Task{
		ID: models.TaskID{
			Reference:   id,
			ConnectorID: connectorID,
		},
		ConnectorID: &connectorID,
		Status:      models.TASK_STATUS_PROCESSING,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := e.storage.TasksUpsert(ctx, task); err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	_, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       id,
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunDeleteUserConnection,
		workflow.DeleteUserConnection{
			TaskID:       task.ID,
			ConnectorID:  connectorID,
			PsuID:        psuID,
			ConnectionID: connectionID,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	return task, nil
}

func (e *engine) CreateUserLink(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, idempotencyKey *uuid.UUID, ClientRedirectURL *string) (models.Task, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.CreateUserLink")
	defer span.End()

	id := models.TaskIDReference(fmt.Sprintf("create-user-link-%s", e.stack), connectorID, psuID.String())
	now := time.Now().UTC()
	task := models.Task{
		ID: models.TaskID{
			Reference:   id,
			ConnectorID: connectorID,
		},
		ConnectorID: &connectorID,
		Status:      models.TASK_STATUS_PROCESSING,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := e.storage.TasksUpsert(ctx, task); err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	_, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       id,
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunCreateUserLink,
		workflow.CreateUserLink{
			TaskID:            task.ID,
			ConnectorID:       connectorID,
			PsuID:             psuID,
			IdempotencyKey:    idempotencyKey,
			ClientRedirectURL: ClientRedirectURL,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	return task, nil
}

func (e *engine) UpdateUserLink(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string, idempotencyKey *uuid.UUID, ClientRedirectURL *string) (models.Task, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.UpdateUserLink")
	defer span.End()

	id := models.TaskIDReference(fmt.Sprintf("update-user-link-%s", e.stack), connectorID, psuID.String())
	now := time.Now().UTC()
	task := models.Task{
		ID: models.TaskID{
			Reference:   id,
			ConnectorID: connectorID,
		},
		ConnectorID: &connectorID,
		Status:      models.TASK_STATUS_PROCESSING,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := e.storage.TasksUpsert(ctx, task); err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	_, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       id,
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
		},
		workflow.RunUpdateUserLink,
		workflow.UpdateUserLink{
			TaskID:            task.ID,
			ConnectorID:       connectorID,
			PsuID:             psuID,
			ConnectionID:      connectionID,
			IdempotencyKey:    idempotencyKey,
			ClientRedirectURL: ClientRedirectURL,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return models.Task{}, err
	}

	return task, nil
}

func (e *engine) CompleteUserLink(ctx context.Context, connectorID models.ConnectorID, attemptID uuid.UUID, httpCallInformation models.HTTPCallInformation) error {
	ctx, span := otel.Tracer().Start(ctx, "engine.CompleteUserLink")
	defer span.End()

	ctx = context.WithoutCancel(ctx)
	e.wg.Add(1)
	defer e.wg.Done()

	id := models.TaskIDReference(fmt.Sprintf("complete-user-link-%s", e.stack), connectorID, attemptID.String())
	_, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       id,
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunCompleteUserLink,
		workflow.CompleteUserLink{
			HTTPCallInformation: httpCallInformation,
			ConnectorID:         connectorID,
			AttemptID:           attemptID,
		},
	)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	return nil
}

func (e *engine) HandleWebhook(ctx context.Context, url string, urlPath string, webhook models.Webhook) error {
	ctx, span := otel.Tracer().Start(ctx, "engine.HandleWebhook")
	defer span.End()

	ctx = context.WithoutCancel(ctx)
	e.wg.Add(1)
	defer e.wg.Done()

	webhook, config, err := e.verifyAndTrimWebhook(ctx, urlPath, webhook)
	if err != nil {
		return err
	}

	if _, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("webhook-%s-%s-%s", e.stack, webhook.ConnectorID.String(), webhook.ID),
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunHandleWebhooks,
		workflow.HandleWebhooks{
			ConnectorID: webhook.ConnectorID,
			URL:         url,
			URLPath:     urlPath,
			Webhook:     webhook,
			Config:      config,
		},
	); err != nil {
		otel.RecordError(span, err)
		return err
	}

	return nil
}

func (e *engine) verifyAndTrimWebhook(ctx context.Context, urlPath string, webhook models.Webhook) (models.Webhook, *models.WebhookConfig, error) {
	plugin, err := e.plugins.Get(webhook.ConnectorID)
	if err != nil {
		return models.Webhook{}, nil, err
	}

	configs, err := e.storage.WebhooksConfigsGetFromConnectorID(ctx, webhook.ConnectorID)
	if err != nil {
		return models.Webhook{}, nil, err
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
		return models.Webhook{}, nil, errors.New("webhook config not found")
	}

	config.FullURL, err = url.JoinPath(e.stackPublicURL, "/api/payments/v3", urlPath)
	if err != nil {
		return models.Webhook{}, nil, err
	}

	verifyResponse, err := plugin.VerifyWebhook(
		ctx,
		models.VerifyWebhookRequest{
			Webhook: models.PSPWebhook{
				BasicAuth:   webhook.BasicAuth,
				QueryValues: webhook.QueryValues,
				Headers:     webhook.Headers,
				Body:        webhook.Body,
			},
			Config: config,
		},
	)
	if err != nil {
		return models.Webhook{}, nil, err
	}

	webhook.IdempotencyKey = verifyResponse.WebhookIdempotencyKey

	webhook, err = e.trimWebhook(ctx, plugin, config, webhook)
	if err != nil {
		return models.Webhook{}, nil, err
	}

	return webhook, config, nil
}

func (e *engine) trimWebhook(ctx context.Context, plugin models.Plugin, config *models.WebhookConfig, webhook models.Webhook) (models.Webhook, error) {
	trimmedWebhook, err := plugin.TrimWebhook(ctx, models.TrimWebhookRequest{
		Webhook: models.PSPWebhook{
			BasicAuth:   webhook.BasicAuth,
			QueryValues: webhook.QueryValues,
			Headers:     webhook.Headers,
			Body:        webhook.Body,
		},
		Config: config,
	})
	if err != nil {
		return models.Webhook{}, err
	}

	webhook.BasicAuth = trimmedWebhook.Webhook.BasicAuth
	webhook.QueryValues = trimmedWebhook.Webhook.QueryValues
	webhook.Headers = trimmedWebhook.Webhook.Headers
	webhook.Body = trimmedWebhook.Webhook.Body

	return webhook, nil
}

func (e *engine) CreatePool(ctx context.Context, pool models.Pool) error {
	ctx, span := otel.Tracer().Start(ctx, "engine.CreatePool")
	defer span.End()

	eg, groupCtx := errgroup.WithContext(ctx)
	for _, accountID := range pool.PoolAccounts {
		aID := accountID
		eg.Go(func() error {
			acc, err := e.storage.AccountsGet(groupCtx, aID)
			if err != nil {
				return err
			}
			if acc.Type != models.ACCOUNT_TYPE_INTERNAL {
				return fmt.Errorf("account %s is not an internal account: %w", aID, ErrValidation)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		otel.RecordError(span, err)
		return err
	}

	if err := e.storage.PoolsUpsert(ctx, pool); err != nil {
		otel.RecordError(span, err)
		return err
	}

	// Do not wait for sending of events
	_, err := e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("pools-creation-%s-%s", e.stack, pool.ID.String()),
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
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

	// Check if the account exists and if it's an INTERNAL account
	account, err := e.storage.AccountsGet(ctx, accountID)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	if account.Type != models.ACCOUNT_TYPE_INTERNAL {
		return fmt.Errorf("account %s is not an internal account: %w", accountID, ErrValidation)
	}

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
			ID:                                       fmt.Sprintf("pools-add-account-%s-%s-%s", e.stack, pool.ID.String(), accountID.String()),
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
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
			ID:                                       fmt.Sprintf("pools-remove-account-%s-%s-%s", e.stack, pool.ID.String(), accountID.String()),
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
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

	deleted, err := e.storage.PoolsDelete(ctx, poolID)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	if deleted {
		// Do not wait for sending of events
		_, err := e.temporalClient.ExecuteWorkflow(
			ctx,
			client.StartWorkflowOptions{
				ID:                                       fmt.Sprintf("pools-deletion-%s-%s", e.stack, poolID.String()),
				TaskQueue:                                GetDefaultTaskQueue(e.stack),
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

	if !connector.ScheduledForDeletion {
		if err := e.plugins.LoadPlugin(
			connector.ID,
			connector.Provider,
			connector.Name,
			config,
			connector.Config,
			false,
		); err != nil {
			return err
		}

		// Launch the workflow
		if _, err := e.launchInstallWorkflow(ctx, connector, config); err != nil {
			return err
		}
	}
	return nil
}

func (e *engine) launchInstallWorkflow(ctx context.Context, connector models.Connector, config models.Config) (client.WorkflowRun, error) {
	return e.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       e.taskIDReferenceFor(IDPrefixConnectorInstall, connector.ID, ""),
			TaskQueue:                                GetDefaultTaskQueue(e.stack),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunInstallConnector,
		workflow.InstallConnector{
			ConnectorID: connector.ID,
			Config:      config,
		},
	)
}

var _ Engine = &engine{}
