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
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/connectors/engine/utils"
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
	ForwardPaymentServiceUser(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) error
	// Delete a payment service user
	DeletePaymentServiceUser(ctx context.Context, psuID uuid.UUID) (models.Task, error)
	// Delete a payment service user on a given connector (PSP)
	DeletePaymentServiceUserConnector(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) (models.Task, error)
	// Delete a payment service user specific connection on the given connector (PSP)
	DeletePaymentServiceUserConnection(ctx context.Context, connectorID models.ConnectorID, psuID uuid.UUID, connectionID string) (models.Task, error)
	// Create a payment service user link on the given connector (PSP).
	CreatePaymentServiceUserLink(ctx context.Context, applicationName string, psuID uuid.UUID, connectorID models.ConnectorID, idempotencyKey *uuid.UUID, ClientRedirectURL *string) (string, string, error)
	// Create a payment service user update link on the given connector (PSP).
	UpdatePaymentServiceUserLink(ctx context.Context, applicationName string, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string, idempotencyKey *uuid.UUID, ClientRedirectURL *string) (string, string, error)
	// Complete a payment service user link on the given connector (PSP).
	CompletePaymentServiceUserLink(ctx context.Context, connectorID models.ConnectorID, attemptID uuid.UUID, httpCallInformation models.HTTPCallInformation) error

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

	// Called when the engine is starting, to start all the connectors.
	OnStart(ctx context.Context) error
	// Called when the engine is stopping, to stop all the connectors.
	OnStop(ctx context.Context)
}

type engine struct {
	logger logging.Logger

	temporalClient client.Client
	storage        storage.Storage

	// connectors is only really present in engine to allow validation of plugin configs prior to insert into the DB
	// other plugin-side work should be performed inside workers and not directly in the engine since we don't
	// have a listener function that checks for plugin updates or installs
	connectors connectors.Manager

	stack          string
	stackPublicURL string

	wg sync.WaitGroup
}

func New(
	logger logging.Logger,
	temporalClient client.Client,
	storage storage.Storage,
	connectors connectors.Manager,
	stack string,
	stackPublicURL string,
) Engine {
	return &engine{
		logger:         logger,
		temporalClient: temporalClient,
		storage:        storage,
		connectors:     connectors,
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
		return models.ConnectorID{}, errorsutils.NewWrappedError(err, ErrValidation)
	}

	if err := config.Validate(); err != nil {
		otel.RecordError(span, err)
		return models.ConnectorID{}, errorsutils.NewWrappedError(err, ErrValidation)
	}

	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  provider,
	}
	validatedConfig, err := e.connectors.Load(connectorID, provider, config.Name, config, rawConfig, false)
	if err != nil {
		otel.RecordError(span, err)
		if _, ok := err.(validator.ValidationErrors); ok || errors.Is(err, models.ErrInvalidConfig) {
			return models.ConnectorID{}, errorsutils.NewWrappedError(err, ErrValidation)
		}
		return models.ConnectorID{}, err
	}

	connector := models.Connector{
		ConnectorBase: models.ConnectorBase{
			ID:        connectorID,
			Name:      config.Name,
			CreatedAt: time.Now().UTC(),
			Provider:  provider,
		},
		Config: validatedConfig,
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

	validatedConfig, err := e.connectors.Load(connector.ID, connector.Provider, config.Name, config, rawConfig, true)
	if err != nil {
		otel.RecordError(span, err)
		if _, ok := err.(validator.ValidationErrors); ok || errors.Is(err, models.ErrInvalidConfig) {
			return errorsutils.NewWrappedError(err, ErrValidation)
		}
		return err
	}
	connector.Name = config.Name
	connector.Config = validatedConfig

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

func (e *engine) ForwardPaymentServiceUser(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) error {
	ctx, span := otel.Tracer().Start(ctx, "engine.ForwardPaymentServiceUser")
	defer span.End()

	_, err := e.storage.OpenBankingForwardedUserGet(ctx, psuID, connectorID)
	switch {
	case err == nil:
		err := fmt.Errorf("user already exists on this connector: %w", ErrValidation)
		otel.RecordError(span, err)
		return err
	case err != nil && !errors.Is(err, storage.ErrNotFound):
		otel.RecordError(span, err)
		return err
	default:
	}

	psu, err := e.storage.PaymentServiceUsersGet(ctx, psuID)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	detachedCtx := context.WithoutCancel(ctx)
	e.wg.Add(1)
	defer e.wg.Done()

	plugin, err := e.connectors.Get(connectorID)
	if err != nil {
		otel.RecordError(span, err)
		if errors.Is(err, connectors.ErrNotFound) {
			return fmt.Errorf("connector %w", ErrNotFound)
		}
		return err
	}

	resp, err := plugin.CreateUser(ctx, models.CreateUserRequest{
		PaymentServiceUser: models.ToPSPPaymentServiceUser(psu),
	})
	if err != nil {
		otel.RecordError(span, err)
		return handlePluginErrors(err)
	}

	openBankingForwardedUser := models.OpenBankingForwardedUser{
		ConnectorID: connectorID,
		AccessToken: resp.PermanentToken,
		PSPUserID:   resp.PSPUserID,
		Metadata:    resp.Metadata,
	}

	err = e.storage.OpenBankingForwardedUserUpsert(detachedCtx, psuID, openBankingForwardedUser)
	if err != nil {
		otel.RecordError(span, err)
		return err
	}

	return nil
}

func (e *engine) DeletePaymentServiceUser(ctx context.Context, psuID uuid.UUID) (models.Task, error) {
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
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunDeletePSU,
		workflow.DeletePSU{
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

func (e *engine) DeletePaymentServiceUserConnector(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) (models.Task, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.DeleteUserConnector")
	defer span.End()

	id := models.TaskIDReference(fmt.Sprintf("delete-user-connector-%s", e.stack), connectorID, psuID.String())
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
		workflow.RunDeletePSUConnector,
		workflow.DeletePSUConnector{
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

func (e *engine) DeletePaymentServiceUserConnection(ctx context.Context, connectorID models.ConnectorID, psuID uuid.UUID, connectionID string) (models.Task, error) {
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
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
			WorkflowExecutionErrorWhenAlreadyStarted: false,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: e.stack,
			},
		},
		workflow.RunDeleteConnection,
		workflow.DeleteConnection{
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

func (e *engine) CreatePaymentServiceUserLink(ctx context.Context, applicationName string, psuID uuid.UUID, connectorID models.ConnectorID, idempotencyKey *uuid.UUID, ClientRedirectURL *string) (string, string, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.CreateUserLink")
	defer span.End()

	plugin, err := e.connectors.Get(connectorID)
	if err != nil {
		otel.RecordError(span, err)
		if errors.Is(err, connectors.ErrNotFound) {
			return "", "", fmt.Errorf("connector %w", ErrNotFound)
		}
		return "", "", err
	}

	psu, err := e.storage.PaymentServiceUsersGet(ctx, psuID)
	if err != nil {
		otel.RecordError(span, err)
		return "", "", err
	}

	openBankingForwardedUser, err := e.storage.OpenBankingForwardedUserGet(ctx, psuID, connectorID)
	if err != nil {
		otel.RecordError(span, err)
		return "", "", err
	}

	id := uuid.New()
	if idempotencyKey != nil {
		id = *idempotencyKey
	}

	attempt := models.OpenBankingConnectionAttempt{
		ID:          id,
		PsuID:       psuID,
		ConnectorID: connectorID,
		CreatedAt:   time.Now().UTC(),
		Status:      models.OpenBankingConnectionAttemptStatusPending,
		State: models.CallbackState{
			Randomized: uuid.New().String(),
			AttemptID:  id,
		},
		ClientRedirectURL: ClientRedirectURL,
	}

	detachedCtx := context.WithoutCancel(ctx)
	e.wg.Add(1)
	defer e.wg.Done()

	err = e.storage.OpenBankingConnectionAttemptsUpsert(
		detachedCtx,
		attempt,
	)
	if err != nil {
		otel.RecordError(span, err)
		return "", "", err
	}

	webhookBaseURL, err := utils.GetWebhookBaseURL(e.stackPublicURL, connectorID)
	if err != nil {
		return "", "", fmt.Errorf("joining webhook base URL: %w", err)
	}

	formanceRedirectURL, err := utils.GetFormanceRedirectURL(e.stackPublicURL, connectorID)
	if err != nil {
		return "", "", fmt.Errorf("joining formance redirect URI: %w", err)
	}

	resp, err := plugin.CreateUserLink(detachedCtx, models.CreateUserLinkRequest{
		ApplicationName:          applicationName,
		AttemptID:                attempt.ID.String(),
		PaymentServiceUser:       models.ToPSPPaymentServiceUser(psu),
		OpenBankingForwardedUser: openBankingForwardedUser,
		ClientRedirectURL:        ClientRedirectURL,
		FormanceRedirectURL:      &formanceRedirectURL,
		CallBackState:            attempt.State.String(),
		WebhookBaseURL:           webhookBaseURL,
	})
	if err != nil {
		otel.RecordError(span, err)
		return "", "", handlePluginErrors(err)
	}

	attempt.TemporaryToken = resp.TemporaryLinkToken
	err = e.storage.OpenBankingConnectionAttemptsUpsert(
		detachedCtx,
		attempt,
	)
	if err != nil {
		return "", "", err
	}

	return attempt.ID.String(), resp.Link, nil
}

func (e *engine) UpdatePaymentServiceUserLink(ctx context.Context, applicationName string, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string, idempotencyKey *uuid.UUID, ClientRedirectURL *string) (string, string, error) {
	ctx, span := otel.Tracer().Start(ctx, "engine.UpdateUserLink")
	defer span.End()

	plugin, err := e.connectors.Get(connectorID)
	if err != nil {
		otel.RecordError(span, err)
		if errors.Is(err, connectors.ErrNotFound) {
			return "", "", fmt.Errorf("connector %w", ErrNotFound)
		}
		return "", "", err
	}

	psu, err := e.storage.PaymentServiceUsersGet(ctx, psuID)
	if err != nil {
		otel.RecordError(span, err)
		return "", "", err
	}

	openBankingForwardedUser, err := e.storage.OpenBankingForwardedUserGet(ctx, psuID, connectorID)
	if err != nil {
		otel.RecordError(span, err)
		return "", "", err
	}

	connection, _, err := e.storage.OpenBankingConnectionsGetFromConnectionID(
		ctx,
		connectorID,
		connectionID,
	)
	if err != nil {
		return "", "", err
	}

	id := uuid.New()
	if idempotencyKey != nil {
		id = *idempotencyKey
	}

	attempt := models.OpenBankingConnectionAttempt{
		ID:          id,
		PsuID:       psuID,
		ConnectorID: connectorID,
		CreatedAt:   time.Now().UTC(),
		Status:      models.OpenBankingConnectionAttemptStatusPending,
		State: models.CallbackState{
			Randomized: uuid.New().String(),
			AttemptID:  id,
		},
		ClientRedirectURL: ClientRedirectURL,
	}

	detachedCtx := context.WithoutCancel(ctx)
	e.wg.Add(1)
	defer e.wg.Done()

	err = e.storage.OpenBankingConnectionAttemptsUpsert(
		detachedCtx,
		attempt,
	)
	if err != nil {
		otel.RecordError(span, err)
		return "", "", err
	}

	webhookBaseURL, err := utils.GetWebhookBaseURL(e.stackPublicURL, connectorID)
	if err != nil {
		return "", "", fmt.Errorf("joining webhook base URL: %w", err)
	}

	formanceRedirectURL, err := utils.GetFormanceRedirectURL(e.stackPublicURL, connectorID)
	if err != nil {
		return "", "", fmt.Errorf("joining formance redirect URI: %w", err)
	}

	resp, err := plugin.UpdateUserLink(detachedCtx, models.UpdateUserLinkRequest{
		AttemptID:                attempt.ID.String(),
		PaymentServiceUser:       models.ToPSPPaymentServiceUser(psu),
		OpenBankingForwardedUser: openBankingForwardedUser,
		Connection:               connection,
		ApplicationName:          applicationName,
		ClientRedirectURL:        ClientRedirectURL,
		FormanceRedirectURL:      &formanceRedirectURL,
		CallBackState:            attempt.State.String(),
		WebhookBaseURL:           webhookBaseURL,
	})
	if err != nil {
		otel.RecordError(span, err)
		return "", "", handlePluginErrors(err)
	}

	attempt.TemporaryToken = resp.TemporaryLinkToken
	err = e.storage.OpenBankingConnectionAttemptsUpsert(
		detachedCtx,
		attempt,
	)
	if err != nil {
		return "", "", err
	}

	return attempt.ID.String(), resp.Link, nil
}

func (e *engine) CompletePaymentServiceUserLink(ctx context.Context, connectorID models.ConnectorID, attemptID uuid.UUID, httpCallInformation models.HTTPCallInformation) error {
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

	webhooks, config, err := e.verifyAndTrimWebhook(ctx, urlPath, webhook)
	if err != nil {
		return err
	}

	errGroup, groupCtx := errgroup.WithContext(ctx)
	for _, webhook := range webhooks {
		w := webhook
		errGroup.Go(func() error {
			if _, err := e.temporalClient.ExecuteWorkflow(
				groupCtx,
				client.StartWorkflowOptions{
					ID:                                       fmt.Sprintf("webhook-%s-%s-%s", e.stack, w.ConnectorID.String(), w.ID),
					TaskQueue:                                GetDefaultTaskQueue(e.stack),
					WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
					WorkflowExecutionErrorWhenAlreadyStarted: false,
					SearchAttributes: map[string]interface{}{
						workflow.SearchAttributeStack: e.stack,
					},
				},
				workflow.RunHandleWebhooks,
				workflow.HandleWebhooks{
					ConnectorID: w.ConnectorID,
					URL:         url,
					URLPath:     urlPath,
					Webhook:     w,
					Config:      config,
				},
			); err != nil {
				return err
			}
			return nil
		})
	}

	if err := errGroup.Wait(); err != nil {
		otel.RecordError(span, err)
		return err
	}

	return nil
}

func (e *engine) verifyAndTrimWebhook(ctx context.Context, urlPath string, webhook models.Webhook) ([]models.Webhook, *models.WebhookConfig, error) {
	plugin, err := e.connectors.Get(webhook.ConnectorID)
	if err != nil {
		return nil, nil, err
	}

	configs, err := e.storage.WebhooksConfigsGetFromConnectorID(ctx, webhook.ConnectorID)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, errors.New("webhook config not found")
	}

	config.FullURL, err = url.JoinPath(e.stackPublicURL, "/api/payments/v3", urlPath)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}

	webhook.IdempotencyKey = verifyResponse.WebhookIdempotencyKey

	webhooks, err := e.trimWebhook(ctx, plugin, config, webhook)
	if err != nil {
		return nil, nil, err
	}

	return webhooks, config, nil
}

func (e *engine) trimWebhook(ctx context.Context, plugin models.Plugin, config *models.WebhookConfig, webhook models.Webhook) ([]models.Webhook, error) {
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
		return nil, err
	}

	webhooks := make([]models.Webhook, 0, len(trimmedWebhook.Webhooks))
	for i, w := range trimmedWebhook.Webhooks {
		var ik *string
		if webhook.IdempotencyKey != nil {
			ik = pointer.For(fmt.Sprintf("%s-%d", *webhook.IdempotencyKey, i))
		}
		webhooks = append(webhooks, models.Webhook{
			ID:             uuid.New().String(),
			ConnectorID:    webhook.ConnectorID,
			IdempotencyKey: ik,
			BasicAuth:      w.BasicAuth,
			QueryValues:    w.QueryValues,
			Headers:        w.Headers,
			Body:           w.Body,
		})
	}

	return webhooks, nil
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
	if err := e.storage.ListenConnectorsChanges(ctx, storage.HandlerConnectorsChanges{
		storage.ConnectorChangesInsert: e.onInsertPlugin,
		storage.ConnectorChangesUpdate: e.onUpdatePlugin,
		storage.ConnectorChangesDelete: e.onDeletePlugin,
	}); err != nil {
		return fmt.Errorf("failed to start engine connector changes listener: %w", err)
	}

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

func (w *engine) onInsertPlugin(ctx context.Context, connectorID models.ConnectorID) error {
	w.logger.Debugf("api got insert notification for %q", connectorID.String())
	connector, err := w.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		return err
	}

	config := models.DefaultConfig()
	if err := json.Unmarshal(connector.Config, &config); err != nil {
		return err
	}

	_, err = w.connectors.Load(
		connector.ID,
		connector.Provider,
		connector.Name,
		config,
		connector.Config,
		false)
	if err != nil {
		return err
	}

	// No need to launch the install workflow here

	return nil
}

func (e *engine) onUpdatePlugin(ctx context.Context, connectorID models.ConnectorID) error {
	e.logger.Debugf("api got update notification for %q", connectorID.String())
	connector, err := e.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return e.onDeletePlugin(ctx, connectorID)
		}
		return err
	}

	if connector.ScheduledForDeletion {
		// if we're deleting the plugin no other changes matter
		return nil
	}

	config := models.DefaultConfig()
	if err := json.Unmarshal(connector.Config, &config); err != nil {
		return err
	}

	_, err = e.connectors.Load(
		connector.ID,
		connector.Provider,
		connector.Name,
		config,
		connector.Config,
		true,
	)
	if err != nil {
		e.logger.Errorf("failed to register plugin after update to connector %q: %v", connector.ID.String(), err)
		return err
	}
	return nil
}

func (e *engine) onDeletePlugin(ctx context.Context, connectorID models.ConnectorID) error {
	e.logger.Debugf("api got delete notification for %q", connectorID.String())
	e.connectors.Unload(connectorID)
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
		if _, err := e.connectors.Load(
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
