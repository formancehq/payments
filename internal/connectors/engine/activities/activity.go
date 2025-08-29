package activities

import (
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	temporalworker "github.com/formancehq/go-libs/v3/temporal"
	"github.com/formancehq/payments/internal/connectors/engine/plugins"
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

	plugins plugins.Plugins
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
			Name: "StorageBalancesDelete",
			Func: a.StorageBalancesDelete,
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
			Name: "StoragePaymentInitiationIDsListFromPaymentID",
			Func: a.StoragePaymentInitiationIDsListFromPaymentID,
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
			Name: "StoragePSUBankBridgesStore",
			Func: a.StoragePSUBankBridgesStore,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePSUBankBridgesGet",
			Func: a.StoragePSUBankBridgesGet,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePSUBankBridgesGetByPSPUserID",
			Func: a.StoragePSUBankBridgesGetByPSPUserID,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePSUBankBridgesDelete",
			Func: a.StoragePSUBankBridgesDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePSUBankBridgesList",
			Func: a.StoragePSUBankBridgesList,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePSUBankBridgeConnectionsStore",
			Func: a.StoragePSUBankBridgeConnectionsStore,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePSUBankBridgeConnectionsLastUpdatedAtUpdate",
			Func: a.StoragePSUBankBridgeConnectionsLastUpdatedAtUpdate,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePSUBankBridgeConnectionDelete",
			Func: a.StoragePSUBankBridgeConnectionDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePSUBankBridgeConnectionAttemptsStore",
			Func: a.StoragePSUBankBridgeConnectionAttemptsStore,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePSUBankBridgeConnectionAttemptsUpdateStatus",
			Func: a.StoragePSUBankBridgeConnectionAttemptsUpdateStatus,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePSUBankBridgeConnectionAttemptsGet",
			Func: a.StoragePSUBankBridgeConnectionAttemptsGet,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePSUBankBridgeConnectionsGetFromConnectionID",
			Func: a.StoragePSUBankBridgeConnectionsGetFromConnectionID,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendAccount",
			Func: a.EventsSendAccount,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendBalance",
			Func: a.EventsSendBalance,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendBankAccount",
			Func: a.EventsSendBankAccount,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendConnectorReset",
			Func: a.EventsSendConnectorReset,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendPayment",
			Func: a.EventsSendPayment,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendPaymentDeleted",
			Func: a.EventsSendPaymentDeleted,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendPoolCreation",
			Func: a.EventsSendPoolCreation,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendPoolDeletion",
			Func: a.EventsSendPoolDeletion,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendPaymentInitiation",
			Func: a.EventsSendPaymentInitiation,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendPaymentInitiationAdjustment",
			Func: a.EventsSendPaymentInitiationAdjustment,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendPaymentInitiationRelatedPayment",
			Func: a.EventsSendPaymentInitiationRelatedPayment,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendUserPendingDisconnect",
			Func: a.EventsSendUserPendingDisconnect,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendUserConnectionDisconnected",
			Func: a.EventsSendUserConnectionDisconnected,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendUserConnectionReconnected",
			Func: a.EventsSendUserConnectionReconnected,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendUserDisconnected",
			Func: a.EventsSendUserDisconnected,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendUserLinkStatus",
			Func: a.EventsSendUserLinkStatus,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendUserConnectionDataSynced",
			Func: a.EventsSendUserConnectionDataSynced,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendTaskUpdated",
			Func: a.EventsSendTaskUpdated,
		}).
		Append(temporalworker.Definition{
			Name: "TemporalScheduleCreate",
			Func: a.TemporalScheduleCreate,
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
		})
}

func New(
	logger logging.Logger,
	temporalClient client.Client,
	storage storage.Storage,
	events *events.Events,
	plugins plugins.Plugins,
	rateLimitingRetryDelay time.Duration,
) Activities {
	return Activities{
		logger:                 logger,
		temporalClient:         temporalClient,
		storage:                storage,
		plugins:                plugins,
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
