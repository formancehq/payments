package activities

import (
	"errors"

	temporalworker "github.com/formancehq/go-libs/v2/temporal"
	"github.com/formancehq/payments/internal/connectors/engine/plugins"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type Activities struct {
	storage        storage.Storage
	events         *events.Events
	temporalClient client.Client

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
			Name: "PluginTranslateWebhook",
			Func: a.PluginTranslateWebhook,
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
			Name: "StorageAccountsDelete",
			Func: a.StorageAccountsDelete,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentsStore",
			Func: a.StoragePaymentsStore,
		}).
		Append(temporalworker.Definition{
			Name: "StoragePaymentsDelete",
			Func: a.StoragePaymentsDelete,
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
			Name: "StoragePaymentInitiationsAdjusmentsIfPredicateStore",
			Func: a.StoragePaymentInitiationsAdjusmentsIfPredicateStore,
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
			Name: "EventsSendPoolCreation",
			Func: a.EventsSendPoolCreation,
		}).
		Append(temporalworker.Definition{
			Name: "EventsSendPoolDeletion",
			Func: a.EventsSendPoolDeletion,
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

func New(temporalClient client.Client, storage storage.Storage, events *events.Events, plugins plugins.Plugins) Activities {
	return Activities{
		temporalClient: temporalClient,
		storage:        storage,
		plugins:        plugins,
		events:         events,
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
