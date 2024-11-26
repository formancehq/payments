package storage

import (
	"context"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

//go:generate mockgen -source storage.go -destination storage_generated.go -package storage . Storage
type Storage interface {
	// Close closes the storage.
	Close() error

	// Accounts
	AccountsUpsert(ctx context.Context, accounts []models.Account) error
	AccountsGet(ctx context.Context, id models.AccountID) (*models.Account, error)
	AccountsList(ctx context.Context, q ListAccountsQuery) (*bunpaginate.Cursor[models.Account], error)
	AccountsDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error

	// Balances
	BalancesUpsert(ctx context.Context, balances []models.Balance) error
	BalancesDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error
	BalancesList(ctx context.Context, q ListBalancesQuery) (*bunpaginate.Cursor[models.Balance], error)
	BalancesGetAt(ctx context.Context, accountID models.AccountID, at time.Time) ([]*models.Balance, error)

	// Bank Accounts
	BankAccountsUpsert(ctx context.Context, bankAccount models.BankAccount) error
	BankAccountsUpdateMetadata(ctx context.Context, id uuid.UUID, metadata map[string]string) error
	BankAccountsGet(ctx context.Context, id uuid.UUID, expand bool) (*models.BankAccount, error)
	BankAccountsList(ctx context.Context, q ListBankAccountsQuery) (*bunpaginate.Cursor[models.BankAccount], error)
	BankAccountsAddRelatedAccount(ctx context.Context, relatedAccount models.BankAccountRelatedAccount) error
	BankAccountsDeleteRelatedAccountFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error

	// Connectors
	ListenConnectorsChanges(ctx context.Context, handler HandlerConnectorsChanges) error
	ConnectorsInstall(ctx context.Context, c models.Connector) error
	ConnectorsUninstall(ctx context.Context, id models.ConnectorID) error
	ConnectorsGet(ctx context.Context, id models.ConnectorID) (*models.Connector, error)
	ConnectorsList(ctx context.Context, q ListConnectorsQuery) (*bunpaginate.Cursor[models.Connector], error)
	ConnectorsScheduleForDeletion(ctx context.Context, id models.ConnectorID) error

	// Connector Tasks Tree
	ConnectorTasksTreeUpsert(ctx context.Context, connectorID models.ConnectorID, tasks models.ConnectorTasksTree) error
	ConnectorTasksTreeGet(ctx context.Context, connectorID models.ConnectorID) (*models.ConnectorTasksTree, error)
	ConnectorTasksTreeDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error

	// Events Sent
	EventsSentUpsert(ctx context.Context, event models.EventSent) error
	EventsSentGet(ctx context.Context, id models.EventID) (*models.EventSent, error)
	EventsSentExists(ctx context.Context, id models.EventID) (bool, error)
	EventsSentDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error

	// Payments
	PaymentsUpsert(ctx context.Context, payments []models.Payment) error
	PaymentsUpdateMetadata(ctx context.Context, id models.PaymentID, metadata map[string]string) error
	PaymentsGet(ctx context.Context, id models.PaymentID) (*models.Payment, error)
	PaymentsList(ctx context.Context, q ListPaymentsQuery) (*bunpaginate.Cursor[models.Payment], error)
	PaymentsDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error

	// Payment Initiations
	PaymentInitiationsUpsert(ctx context.Context, pi models.PaymentInitiation, adjustments ...models.PaymentInitiationAdjustment) error
	PaymentInitiationsUpdateMetadata(ctx context.Context, piID models.PaymentInitiationID, metadata map[string]string) error
	PaymentInitiationsGet(ctx context.Context, piID models.PaymentInitiationID) (*models.PaymentInitiation, error)
	PaymentInitiationsDelete(ctx context.Context, piID models.PaymentInitiationID) error
	PaymentInitiationsDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error
	PaymentInitiationsList(ctx context.Context, q ListPaymentInitiationsQuery) (*bunpaginate.Cursor[models.PaymentInitiation], error)
	PaymentInitiationIDsListFromPaymentID(ctx context.Context, id models.PaymentID) ([]models.PaymentInitiationID, error)

	// Payment Initiation Adjustments
	PaymentInitiationAdjustmentsUpsert(ctx context.Context, adj models.PaymentInitiationAdjustment) error
	PaymentInitiationAdjustmentsUpsertIfPredicate(ctx context.Context, adj models.PaymentInitiationAdjustment, predicate func(models.PaymentInitiationAdjustment) bool) (bool, error)
	PaymentInitiationAdjustmentsGet(ctx context.Context, id models.PaymentInitiationAdjustmentID) (*models.PaymentInitiationAdjustment, error)
	PaymentInitiationAdjustmentsList(ctx context.Context, piID models.PaymentInitiationID, q ListPaymentInitiationAdjustmentsQuery) (*bunpaginate.Cursor[models.PaymentInitiationAdjustment], error)

	// Payment Initiation Related Payments
	PaymentInitiationRelatedPaymentsUpsert(ctx context.Context, piID models.PaymentInitiationID, pID models.PaymentID, createdAt time.Time) error
	PaymentInitiationRelatedPaymentsList(ctx context.Context, piID models.PaymentInitiationID, q ListPaymentInitiationRelatedPaymentsQuery) (*bunpaginate.Cursor[models.Payment], error)

	// Payment Initiation Reversals
	PaymentInitiationReversalsUpsert(ctx context.Context, pir models.PaymentInitiationReversal, reversalAdjustments []models.PaymentInitiationReversalAdjustment) error
	PaymentInitiationReversalsGet(ctx context.Context, id models.PaymentInitiationReversalID) (*models.PaymentInitiationReversal, error)
	PaymentInitiationReversalsDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error
	PaymentInitiationReversalsList(ctx context.Context, q ListPaymentInitiationReversalsQuery) (*bunpaginate.Cursor[models.PaymentInitiationReversal], error)

	// Payment Initiation Reversal Adjustments
	PaymentInitiationReversalAdjustmentsUpsert(ctx context.Context, adj models.PaymentInitiationReversalAdjustment) error
	PaymentInitiationReversalAdjustmentsGet(ctx context.Context, id models.PaymentInitiationReversalAdjustmentID) (*models.PaymentInitiationReversalAdjustment, error)
	PaymentInitiationReversalAdjustmentsList(ctx context.Context, piID models.PaymentInitiationReversalID, q ListPaymentInitiationReversalAdjustmentsQuery) (*bunpaginate.Cursor[models.PaymentInitiationReversalAdjustment], error)

	// Pools
	PoolsUpsert(ctx context.Context, pool models.Pool) error
	PoolsGet(ctx context.Context, id uuid.UUID) (*models.Pool, error)
	PoolsDelete(ctx context.Context, id uuid.UUID) error
	PoolsAddAccount(ctx context.Context, id uuid.UUID, accountID models.AccountID) error
	PoolsRemoveAccount(ctx context.Context, id uuid.UUID, accountID models.AccountID) error
	PoolsList(ctx context.Context, q ListPoolsQuery) (*bunpaginate.Cursor[models.Pool], error)

	// Schedules
	SchedulesUpsert(ctx context.Context, schedule models.Schedule) error
	SchedulesList(ctx context.Context, q ListSchedulesQuery) (*bunpaginate.Cursor[models.Schedule], error)
	SchedulesGet(ctx context.Context, id string, connectorID models.ConnectorID) (*models.Schedule, error)
	SchedulesDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error
	SchedulesDelete(ctx context.Context, id string) error

	// State
	StatesUpsert(ctx context.Context, state models.State) error
	StatesGet(ctx context.Context, id models.StateID) (models.State, error)
	StatesDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error

	// Tasks
	TasksUpsert(ctx context.Context, task models.Task) error
	TasksGet(ctx context.Context, id models.TaskID) (*models.Task, error)
	TasksDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error

	// Webhooks Configs
	WebhooksConfigsUpsert(ctx context.Context, webhooksConfigs []models.WebhookConfig) error
	WebhooksConfigsGet(ctx context.Context, name string, connectorID models.ConnectorID) (*models.WebhookConfig, error)
	WebhooksConfigsGetFromConnectorID(ctx context.Context, connectorID models.ConnectorID) ([]models.WebhookConfig, error)
	WebhooksConfigsDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error

	// Webhooks
	WebhooksInsert(ctx context.Context, webhook models.Webhook) error
	WebhooksGet(ctx context.Context, id string) (models.Webhook, error)
	WebhooksDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error

	// Workflow Instances
	InstancesUpsert(ctx context.Context, instance models.Instance) error
	InstancesUpdate(ctx context.Context, instance models.Instance) error
	InstancesGet(ctx context.Context, id string, scheduleID string, connectorID models.ConnectorID) (*models.Instance, error)
	InstancesList(ctx context.Context, q ListInstancesQuery) (*bunpaginate.Cursor[models.Instance], error)
	InstancesDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error
}

const encryptionOptions = "compress-algo=1, cipher-algo=aes256"

type store struct {
	logger              logging.Logger
	db                  *bun.DB
	configEncryptionKey string

	conns   []bun.Conn
	rwMutex sync.RWMutex
}

func newStorage(logger logging.Logger, db *bun.DB, configEncryptionKey string) Storage {
	return &store{
		logger:              logger,
		db:                  db,
		configEncryptionKey: configEncryptionKey,
	}
}

func (s *store) Close() error {
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()

	if err := s.db.Close(); err != nil {
		return err
	}

	for _, conn := range s.conns {
		if err := conn.Close(); err != nil {
			return err
		}
	}

	return nil
}
