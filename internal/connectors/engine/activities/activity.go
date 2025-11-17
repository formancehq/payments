package activities

import (
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	temporalworker "github.com/formancehq/go-libs/v3/temporal"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type Activities struct {
	logger         logging.Logger
	storage        storage.Storage
	events         *events.Events
	temporalClient client.Client

	rateLimitingRetryDelay time.Duration

	connectors connectors.Manager
}

func (a Activities) DefinitionSet() temporalworker.DefinitionSet {
	return temporalworker.NewDefinitionSet().
		Append(temporalworker.Definition{
			Name: "PluginInstallConnector",
			Func: a.PluginInstallConnector,
		}).
		Append(temporalworker.Definition{
			Name: "PluginUninstallConnector",
			Func: a.PluginUninstallConnector,
		}).
		Append(temporalworker.Definition{
			Name: "PluginFetchNextAccounts",
			Func: a.PluginFetchNextAccounts,
		}).
		Append(temporalworker.Definition{
			Name: "PluginFetchNextBalances",
			Func: a.PluginFetchNextBalances,
		}).
		Append(temporalworker.Definition{
			Name: "PluginFetchNextExternalAccounts",
			Func: a.PluginFetchNextExternalAccounts,
		}).
		Append(temporalworker.Definition{
			Name: "PluginFetchNextPayments",
			Func: a.PluginFetchNextPayments,
		}).
		Append(temporalworker.Definition{
			Name: "PluginFetchNextOthers",
			Func: a.PluginFetchNextOthers,
		}).
		Append(temporalworker.Definition{
			Name: "PluginCreateBankAccount",
			Func: a.PluginCreateBankAccount,
		}).
		Append(temporalworker.Definition{
			Name: "PluginCreateTransfert",
			Func: a.PluginCreateTransfer,
		}).
		Append(temporalworker.Definition{
			Name: "PluginReverseTransfer",
			Func: a.PluginReverseTransfer,
		}).
		Append(temporalworker.Definition{
			Name: "PluginPollTransferStatus",
			Func: a.PluginPollTransferStatus,
		}).
		Append(temporalworker.Definition{
			Name: "PluginCreatePayout",
			Func: a.PluginCreatePayout,
		}).
		Append(temporalworker.Definition{
			Name: "PluginReversePayout",
			Func: a.PluginReversePayout,
		}).
		Append(temporalworker.Definition{
			Name: "PluginPollPayoutStatus",
			Func: a.PluginPollPayoutStatus,
		}).
		Append(temporalworker.Definition{
			Name: "PluginCreateWebhooks",
			Func: a.PluginCreateWebhooks,
		}).
		Append(temporalworker.Definition{
			Name: "PluginVerifyWebhook",
			Func: a.PluginVerifyWebhook,
		}).
		Append(temporalworker.Definition{
			Name: "PluginTranslateWebhook",
			Func: a.PluginTranslateWebhook,
		}).
		Append(temporalworker.Definition{
			Name: "PluginCreateUser",
			Func: a.PluginCreateUser,
		}).
		Append(temporalworker.Definition{
			Name: "PluginCreateUserLink",
			Func: a.PluginCreateUserLink,
		}).
		Append(temporalworker.Definition{
			Name: "PluginUpdateUserLink",
			Func: a.PluginUpdateUserLink,
		}).
		Append(temporalworker.Definition{
			Name: "PluginCompleteUserLink",
			Func: a.PluginCompleteUserLink,
		}).
		Append(temporalworker.Definition{
			Name: "PluginDeleteUserConnection",
			Func: a.PluginDeleteUserConnection,
		}).
		Append(temporalworker.Definition{
			Name: "PluginDeleteUser",
			Func: a.PluginDeleteUser,
		}).
		Append(temporalworker.Definition{
			Name: "StorageAccountsStore",
			Func: a.StorageAccountsStore,
		}).
		Append(temporalworker.Definition{
			Name: "StorageAccountsGet",
			Func: a.StorageAccountsGet,
		}).
		Append(temporalworker.Definition{
			Name: "StorageAccountsList",
			Func: a.StorageAccountsList,
		}).
		Append(temporalworker.Definition{
			Name: "StorageAccountsDelete",
			Func: a.StorageAccountsDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StorageAccountsDeleteFromConnectorID",
			Func: a.StorageAccountsDeleteFromConnectorID,
		}).
		Append(temporalworker.Definition{
			Name: "StorageAccountsDeleteFromPSUID",
			Func: a.StorageAccountsDeleteFromPSUID,
		}).
		Append(temporalworker.Definition{
			Name: "StorageAccountsDeleteFromPSUIDAndConnectorID",
			Func: a.StorageAccountsDeleteFromPSUIDAndConnectorID,
		}).
		Append(temporalworker.Definition{
			Name: "StorageAccountsDeleteFromConnectionID",
			Func: a.StorageAccountsDeleteFromConnectionID,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentsStore",
			Func: a.StoragePaymentsStore,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentsList",
			Func: a.StoragePaymentsList,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentsGetByReference",
			Func: a.StoragePaymentsGetByReference,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentsDelete",
			Func: a.StoragePaymentsDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentsDeleteFromConnectorID",
			Func: a.StoragePaymentsDeleteFromConnectorID,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentsDeleteFromReference",
			Func: a.StoragePaymentsDeleteFromReference,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentsDeleteFromAccountID",
			Func: a.StoragePaymentsDeleteFromAccountID,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentsDeleteFromPSUID",
			Func: a.StoragePaymentsDeleteFromPSUID,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentsDeleteFromPSUIDAndConnectorID",
			Func: a.StoragePaymentsDeleteFromPSUIDAndConnectorID,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentsDeleteFromConnectionID",
			Func: a.StoragePaymentsDeleteFromConnectionID,
		}).
		Append(temporalworker.Definition{
			Name: "StorageStatesGet",
			Func: a.StorageStatesGet,
		}).
		Append(temporalworker.Definition{
			Name: "StorageStatesStore",
			Func: a.StorageStatesStore,
		}).
		Append(temporalworker.Definition{
			Name: "StorageStatesDelete",
			Func: a.StorageStatesDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StorageConnectorTasksTreeStore",
			Func: a.StorageConnectorTasksTreeStore,
		}).
		Append(temporalworker.Definition{
			Name: "StorageConnectorTasksTreeDelete",
			Func: a.StorageConnectorTasksTreeDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StorageConnectorsStore",
			Func: a.StorageConnectorsStore,
		}).
		Append(temporalworker.Definition{
			Name: "StorageConnectorsGet",
			Func: a.StorageConnectorsGet,
		}).
		Append(temporalworker.Definition{
			Name: "StorageConnectorsDelete",
			Func: a.StorageConnectorsDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StorageConnectorsScheduleForDeletion",
			Func: a.StorageConnectorsScheduleForDeletion,
		}).
		Append(temporalworker.Definition{
			Name: "StorageSchedulesGet",
			Func: a.StorageSchedulesGet,
		}).
		Append(temporalworker.Definition{
			Name: "StorageSchedulesStore",
			Func: a.StorageSchedulesStore,
		}).
		Append(temporalworker.Definition{
			Name: "StorageSchedulesList",
			Func: a.StorageSchedulesList,
		}).
		Append(temporalworker.Definition{
			Name: "StoreSchedulesDelete",
			Func: a.StorageSchedulesDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StorageSchedulesDeleteFromConnectorID",
			Func: a.StorageSchedulesDeleteFromConnectorID,
		}).
		Append(temporalworker.Definition{
			Name: "StorageInstancesStore",
			Func: a.StorageInstancesStore,
		}).
		Append(temporalworker.Definition{
			Name: "StorageInstancesUpdate",
			Func: a.StorageInstancesUpdate,
		}).
		Append(temporalworker.Definition{
			Name: "StorageInstancesDelete",
			Func: a.StorageInstancesDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StorageBankAccountsDeleteRelatedAccounts",
			Func: a.StorageBankAccountsDeleteRelatedAccounts,
		}).
		Append(temporalworker.Definition{
			Name: "StorageBankAccountsAddRelatedAccount",
			Func: a.StorageBankAccountsAddRelatedAccount,
		}).
		Append(temporalworker.Definition{
			Name: "StorageBankAccountsGet",
			Func: a.StorageBankAccountsGet,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentServiceUsersGet",
			Func: a.StoragePaymentServiceUsersGet,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentServiceUsersDelete",
			Func: a.StoragePaymentServiceUsersDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StorageBalancesStore",
			Func: a.StorageBalancesStore,
		}).
		Append(temporalworker.Definition{
			Name: "StorageWebhooksConfigsStore",
			Func: a.StorageWebhooksConfigsStore,
		}).
		Append(temporalworker.Definition{
			Name: "StorageWebhooksConfigsGet",
			Func: a.StorageWebhooksConfigsGet,
		}).
		Append(temporalworker.Definition{
			Name: "StorageWebhooksConfigsDelete",
			Func: a.StorageWebhooksConfigsDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StorageWebhooksStore",
			Func: a.StorageWebhooksStore,
		}).
		Append(temporalworker.Definition{
			Name: "StorageWebhooksDelete",
			Func: a.StorageWebhooksDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentInitiationGet",
			Func: a.StoragePaymentInitiationsGet,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentInitiationsRelatedPaymentsStore",
			Func: a.StoragePaymentInitiationsRelatedPaymentsStore,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentInitiationsAdjustmentsStore",
			Func: a.StoragePaymentInitiationsAdjustmentsStore,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentInitiationAdjustmentsList",
			Func: a.StoragePaymentInitiationAdjustmentsList,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentInitiationsAdjustmentsIfPredicateStore",
			Func: a.StoragePaymentInitiationsAdjustmentsIfPredicateStore,
		}).
		Append(temporalworker.Definition{ // Only for backward compatibility, we can get rid of it as soon as the above block is released.
			Name: "StoragePaymentInitiationsAdjusmentsIfPredicateStore",
			Func: a.StoragePaymentInitiationsAdjustmentsIfPredicateStore,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentInitiationUpdateFromPayment",
			Func: a.StoragePaymentInitiationUpdateFromPayment,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentInitiationsDelete",
			Func: a.StoragePaymentInitiationsDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentInitiationReversalsGet",
			Func: a.StoragePaymentInitiationReversalsGet,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentInitiationReversalsDelete",
			Func: a.StoragePaymentInitiationReversalsDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentInitiationReversalsAdjustmentsStore",
			Func: a.StoragePaymentInitiationReversalsAdjustmentsStore,
		}).
		Append(temporalworker.Definition{
			Name: "StorageEventsSentStore",
			Func: a.StorageEventsSentStore,
		}).
		Append(temporalworker.Definition{
			Name: "StorageEventsSentDelete",
			Func: a.StorageEventsSentDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StorageEventsSentExists",
			Func: a.StorageEventsSentExists,
		}).
		Append(temporalworker.Definition{
			Name: "StorageTasksStore",
			Func: a.StorageTasksStore,
		}).
		Append(temporalworker.Definition{
			Name: "StorageTasksDelete",
			Func: a.StorageTasksDeleteFromConnectorID,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePoolsRemoveAccountsFromConnectorID",
			Func: a.StoragePoolsRemoveAccountsFromConnectorID,
		}).
		Append(temporalworker.Definition{
			Name: "StorageOpenBankingForwardedUsersStore",
			Func: a.StorageOpenBankingForwardedUsersStore,
		}).
		Append(temporalworker.Definition{
			Name: "StorageOpenBankingForwardedUsersGet",
			Func: a.StorageOpenBankingForwardedUsersGet,
		}).
		Append(temporalworker.Definition{
			Name: "StorageOpenBankingForwardedUsersGetByPSPUserID",
			Func: a.StorageOpenBankingForwardedUsersGetByPSPUserID,
		}).
		Append(temporalworker.Definition{
			Name: "StorageOpenBankingForwardedUsersDelete",
			Func: a.StorageOpenBankingForwardedUsersDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StorageOpenBankingForwardedUsersList",
			Func: a.StorageOpenBankingForwardedUsersList,
		}).
		Append(temporalworker.Definition{
			Name: "StorageOpenBankingConnectionsStore",
			Func: a.StorageOpenBankingConnectionsStore,
		}).
		Append(temporalworker.Definition{
			Name: "StorageOpenBankingConnectionsLastUpdatedAtUpdate",
			Func: a.StorageOpenBankingConnectionsLastUpdatedAtUpdate,
		}).
		Append(temporalworker.Definition{
			Name: "StorageOpenBankingConnectionsDelete",
			Func: a.StorageOpenBankingConnectionsDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StorageOpenBankingConnectionAttemptsStore",
			Func: a.StorageOpenBankingConnectionAttemptsStore,
		}).
		Append(temporalworker.Definition{
			Name: "StorageOpenBankingConnectionAttemptsUpdateStatus",
			Func: a.StorageOpenBankingConnectionAttemptsUpdateStatus,
		}).
		Append(temporalworker.Definition{
			Name: "StorageOpenBankingConnectionAttemptsGet",
			Func: a.StorageOpenBankingConnectionAttemptsGet,
		}).
		Append(temporalworker.Definition{
			Name: "StorageOpenBankingConnectionsGetFromConnectionID",
			Func: a.StorageOpenBankingConnectionsGetFromConnectionID,
		}).
		Append(temporalworker.Definition{
			Name: "SendEvents",
			Func: a.SendEvents,
		}).
		Append(temporalworker.Definition{
			Name: "TemporalScheduleCreate",
			Func: a.TemporalScheduleCreate,
		}).
		Append(temporalworker.Definition{
			Name: "TemporalScheduleUpdatePollingPeriod",
			Func: a.TemporalScheduleUpdatePollingPeriod,
		}).
		Append(temporalworker.Definition{
			Name: "TemporalDeleteSchedule",
			Func: a.TemporalScheduleDelete,
		}).
		Append(temporalworker.Definition{
			Name: "TemporalWorkflowTerminate",
			Func: a.TemporalWorkflowTerminate,
		}).
		Append(temporalworker.Definition{
			Name: "TemporalWorkflowExecutionsList",
			Func: a.TemporalWorkflowExecutionsList,
		}).
		Append(temporalworker.Definition{
			Name: "OutboxPublishPendingEvents",
			Func: a.OutboxPublishPendingEvents,
		}).
		Append(temporalworker.Definition{
			Name: "StorageOutboxEventsInsert",
			Func: a.StorageOutboxEventsInsert,
		}).
		Append(temporalworker.Definition{
			Name: "CreateOutboxPublisherSchedule",
			Func: a.CreateOutboxPublisherSchedule,
		})
}

func New(
	logger logging.Logger,
	temporalClient client.Client,
	storage storage.Storage,
	events *events.Events,
	connectors connectors.Manager,
	rateLimitingRetryDelay time.Duration,
) Activities {
	return Activities{
		logger:                 logger,
		temporalClient:         temporalClient,
		storage:                storage,
		connectors:             connectors,
		events:                 events,
		rateLimitingRetryDelay: rateLimitingRetryDelay,
	}
}

func executeActivity(ctx workflow.Context, activity any, ret any, args ...any) error {
	if err := workflow.ExecuteActivity(ctx, activity, args...).Get(ctx, ret); err != nil {
		var timeoutError *temporal.TimeoutError
		if errors.As(err, &timeoutError) {
			return errors.New(timeoutError.Message())
		}
		return err
	}
	return nil
}
