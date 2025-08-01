package backend

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
)

//go:generate mockgen -source backend.go -destination backend_generated.go -package backend . Backend
type Backend interface {
	// Accounts
	AccountsCreate(ctx context.Context, account models.Account) error
	AccountsList(ctx context.Context, query storage.ListAccountsQuery) (*bunpaginate.Cursor[models.Account], error)
	AccountsGet(ctx context.Context, id models.AccountID) (*models.Account, error)

	// Balances
	BalancesList(ctx context.Context, query storage.ListBalancesQuery) (*bunpaginate.Cursor[models.Balance], error)
	PoolsBalancesAt(ctx context.Context, poolID uuid.UUID, at time.Time) ([]models.AggregatedBalance, error)
	PoolsBalances(ctx context.Context, poolID uuid.UUID) ([]models.AggregatedBalance, error)

	// Bank Accounts
	BankAccountsCreate(ctx context.Context, bankAccount models.BankAccount) error
	BankAccountsGet(ctx context.Context, id uuid.UUID) (*models.BankAccount, error)
	BankAccountsList(ctx context.Context, query storage.ListBankAccountsQuery) (*bunpaginate.Cursor[models.BankAccount], error)
	BankAccountsUpdateMetadata(ctx context.Context, id uuid.UUID, metadata map[string]string) error
	BankAccountsForwardToConnector(ctx context.Context, bankAccountID uuid.UUID, connectorID models.ConnectorID, waitResult bool) (models.Task, error)

	// Connectors
	ConnectorsConfigs() registry.Configs
	ConnectorsConfig(ctx context.Context, connectorID models.ConnectorID) (json.RawMessage, error)
	ConnectorsConfigUpdate(ctx context.Context, connectorID models.ConnectorID, rawConfig json.RawMessage) error
	ConnectorsList(ctx context.Context, query storage.ListConnectorsQuery) (*bunpaginate.Cursor[models.Connector], error)
	ConnectorsInstall(ctx context.Context, provider string, config json.RawMessage) (models.ConnectorID, error)
	ConnectorsUninstall(ctx context.Context, connectorID models.ConnectorID) (models.Task, error)
	ConnectorsReset(ctx context.Context, connectorID models.ConnectorID) (models.Task, error)

	// Payments
	PaymentsCreate(ctx context.Context, payment models.Payment) error
	PaymentsUpdateMetadata(ctx context.Context, id models.PaymentID, metadata map[string]string) error
	PaymentsList(ctx context.Context, query storage.ListPaymentsQuery) (*bunpaginate.Cursor[models.Payment], error)
	PaymentsGet(ctx context.Context, id models.PaymentID) (*models.Payment, error)

	// Payment Initiations
	PaymentInitiationsCreate(ctx context.Context, paymentInitiation models.PaymentInitiation, sendToPSP bool, waitResult bool) (models.Task, error)
	PaymentInitiationsList(ctx context.Context, query storage.ListPaymentInitiationsQuery) (*bunpaginate.Cursor[models.PaymentInitiation], error)
	PaymentInitiationsGet(ctx context.Context, id models.PaymentInitiationID) (*models.PaymentInitiation, error)
	PaymentInitiationsApprove(ctx context.Context, id models.PaymentInitiationID, waitResult bool) (models.Task, error)
	PaymentInitiationsReject(ctx context.Context, id models.PaymentInitiationID) error
	PaymentInitiationsRetry(ctx context.Context, id models.PaymentInitiationID, waitResult bool) (models.Task, error)
	PaymentInitiationsDelete(ctx context.Context, id models.PaymentInitiationID) error

	// Payment Initiation Reversals
	PaymentInitiationReversalsCreate(ctx context.Context, reversal models.PaymentInitiationReversal, waitResult bool) (models.Task, error)

	// Payment Initiation Adjustments
	PaymentInitiationAdjustmentsList(ctx context.Context, id models.PaymentInitiationID, query storage.ListPaymentInitiationAdjustmentsQuery) (*bunpaginate.Cursor[models.PaymentInitiationAdjustment], error)
	PaymentInitiationAdjustmentsListAll(ctx context.Context, id models.PaymentInitiationID) ([]models.PaymentInitiationAdjustment, error)
	PaymentInitiationAdjustmentsGetLast(ctx context.Context, id models.PaymentInitiationID) (*models.PaymentInitiationAdjustment, error)

	// Payment Initiatiion Related Payments
	PaymentInitiationRelatedPaymentsList(ctx context.Context, id models.PaymentInitiationID, query storage.ListPaymentInitiationRelatedPaymentsQuery) (*bunpaginate.Cursor[models.Payment], error)
	PaymentInitiationRelatedPaymentsListAll(ctx context.Context, id models.PaymentInitiationID) ([]models.Payment, error)

	// Payment Service Users
	PaymentServiceUsersCreate(ctx context.Context, psu models.PaymentServiceUser) error
	PaymentServiceUsersGet(ctx context.Context, id uuid.UUID) (*models.PaymentServiceUser, error)
	PaymentServiceUsersList(ctx context.Context, query storage.ListPSUsQuery) (*bunpaginate.Cursor[models.PaymentServiceUser], error)
	PaymentServiceUsersForwardBankAccountToConnector(ctx context.Context, psuID, bankAccountID uuid.UUID, connectorID models.ConnectorID) (models.Task, error)
	PaymentServiceUsersAddBankAccount(ctx context.Context, psuID uuid.UUID, bankAccountID uuid.UUID) error
	PaymentServiceUsersForward(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) error
	PaymentServiceUsersDelete(ctx context.Context, psuID uuid.UUID) (models.Task, error)
	PaymentServiceUsersConnectorDelete(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) (models.Task, error)
	PaymentServiceUsersConnectionsList(ctx context.Context, psuID uuid.UUID, connectorID *models.ConnectorID, query storage.ListPsuBankBridgeConnectionsQuery) (*bunpaginate.Cursor[models.PSUBankBridgeConnection], error)
	PaymentServiceUsersConnectionsDelete(ctx context.Context, connectorID models.ConnectorID, psuID uuid.UUID, connectionID string) (models.Task, error)
	PaymentServiceUsersLinkAttemptsList(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, query storage.ListPSUBankBridgeConnectionAttemptsQuery) (*bunpaginate.Cursor[models.PSUBankBridgeConnectionAttempt], error)
	PaymentServiceUsersLinkAttemptsGet(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, id uuid.UUID) (*models.PSUBankBridgeConnectionAttempt, error)
	PaymentServiceUsersCreateLink(ctx context.Context, ApplicationName string, psuID uuid.UUID, connectorID models.ConnectorID, idempotencyKey *uuid.UUID, ClientRedirectURL *string) (string, string, error)
	PaymentServiceUsersUpdateLink(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string, idempotencyKey *uuid.UUID, ClientRedirectURL *string) (string, string, error)
	PaymentServiceUsersCompleteLinkFlow(ctx context.Context, connectorID models.ConnectorID, httpCallInformation models.HTTPCallInformation) (string, error)

	// Pools
	PoolsCreate(ctx context.Context, pool models.Pool) error
	PoolsGet(ctx context.Context, id uuid.UUID) (*models.Pool, error)
	PoolsList(ctx context.Context, query storage.ListPoolsQuery) (*bunpaginate.Cursor[models.Pool], error)
	PoolsDelete(ctx context.Context, id uuid.UUID) error
	PoolsAddAccount(ctx context.Context, id uuid.UUID, accountID models.AccountID) error
	PoolsRemoveAccount(ctx context.Context, id uuid.UUID, accountID models.AccountID) error

	// Schedules
	SchedulesList(ctx context.Context, query storage.ListSchedulesQuery) (*bunpaginate.Cursor[models.Schedule], error)
	SchedulesGet(ctx context.Context, id string, connectorID models.ConnectorID) (*models.Schedule, error)

	// Tasks
	TaskGet(ctx context.Context, id models.TaskID) (*models.Task, error)

	// Webhooks
	ConnectorsHandleWebhooks(ctx context.Context, url string, urlPath string, webhook models.Webhook) error

	// Workflows Instances
	WorkflowsInstancesList(ctx context.Context, query storage.ListInstancesQuery) (*bunpaginate.Cursor[models.Instance], error)
}
