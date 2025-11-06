package workflow

import (
	"encoding/json"

	"github.com/formancehq/go-libs/v3/logging"
	temporalworker "github.com/formancehq/go-libs/v3/temporal"
	"github.com/formancehq/payments/internal/connectors"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const (
	SearchAttributeScheduleID = "PaymentScheduleID"
	SearchAttributeStack      = "Stack"
)

type FromPayload struct {
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

func (f *FromPayload) GetPayload() json.RawMessage {
	if f == nil {
		return nil
	}
	return f.Payload
}

type Workflow struct {
	temporalNamespace string
	temporalClient    client.Client

	connectors connectors.Manager

	stackPublicURL string
	stack          string

	logger logging.Logger
}

func New(temporalClient client.Client, temporalNamespace string, connectors connectors.Manager, stack string, stackPublicURL string, logger logging.Logger) Workflow {
	return Workflow{
		temporalClient:    temporalClient,
		temporalNamespace: temporalNamespace,
		connectors:        connectors,
		stack:             stack,
		stackPublicURL:    stackPublicURL,
		logger:            logger,
	}
}

func (w Workflow) DefinitionSet() temporalworker.DefinitionSet {
	return temporalworker.NewDefinitionSet().
		Append(temporalworker.Definition{
			Name: RunFetchNextAccounts,
			Func: w.runFetchNextAccounts,
		}).
		Append(temporalworker.Definition{
			Name: RunFetchNextBalances,
			Func: w.runFetchNextBalances,
		}).
		Append(temporalworker.Definition{
			Name: RunFetchNextExternalAccounts,
			Func: w.runFetchNextExternalAccounts,
		}).
		Append(temporalworker.Definition{
			Name: RunFetchNextOthers,
			Func: w.runFetchNextOthers,
		}).
		Append(temporalworker.Definition{
			Name: RunFetchNextPayments,
			Func: w.runFetchNextPayments,
		}).
		Append(temporalworker.Definition{
			Name: RunTerminateSchedules,
			Func: w.runTerminateSchedules,
		}).
		Append(temporalworker.Definition{
			Name: RunTerminateWorkflows,
			Func: w.runTerminateWorkflows,
		}).
		Append(temporalworker.Definition{
			Name: RunInstallConnector,
			Func: w.runInstallConnector,
		}).
		Append(temporalworker.Definition{
			Name: RunResetConnector,
			Func: w.runResetConnector,
		}).
		Append(temporalworker.Definition{
			Name: RunUninstallConnector,
			Func: w.runUninstallConnector,
		}).
		Append(temporalworker.Definition{
			Name: RunCreateBankAccount,
			Func: w.runCreateBankAccount,
		}).
		Append(temporalworker.Definition{
			Name: RunCreatePayout,
			Func: w.runCreatePayout,
		}).
		Append(temporalworker.Definition{
			Name: RunReversePayout,
			Func: w.runReversePayout,
		}).
		Append(temporalworker.Definition{
			Name: RunPollPayout,
			Func: w.runPollPayout,
		}).
		Append(temporalworker.Definition{
			Name: RunCreateTransfer,
			Func: w.runCreateTransfer,
		}).
		Append(temporalworker.Definition{
			Name: RunReverseTransfer,
			Func: w.runReverseTransfer,
		}).
		Append(temporalworker.Definition{
			Name: RunPollTransfer,
			Func: w.runPollTransfer,
		}).
		Append(temporalworker.Definition{
			Name: Run,
			Func: w.run,
		}).
		Append(temporalworker.Definition{
			Name: RunCreateWebhooks,
			Func: w.runCreateWebhooks,
		}).
		Append(temporalworker.Definition{
			Name: RunHandleWebhooks,
			Func: w.runHandleWebhooks,
		}).
		Append(temporalworker.Definition{
			Name: RunStoreWebhookTranslation,
			Func: w.runStoreWebhookTranslation,
		}).
		Append(temporalworker.Definition{
			Name: RunSendEvents,
			Func: w.runSendEvents,
		}).
		Append(temporalworker.Definition{
			Name: RunDeletePSU,
			Func: w.runDeletePSU,
		}).
		Append(temporalworker.Definition{
			Name: RunDeletePSUConnector,
			Func: w.runDeletePSUConnector,
		}).
		Append(temporalworker.Definition{
			Name: RunDeleteConnection,
			Func: w.runDeleteConnection,
		}).
		Append(temporalworker.Definition{
			Name: RunCompleteUserLink,
			Func: w.runCompleteUserLink,
		}).
		Append(temporalworker.Definition{
			Name: RunFetchOpenBankingData,
			Func: w.runFetchOpenBankingData,
		}).
		Append(temporalworker.Definition{
			Name: RunDeleteOpenBankingConnectionData,
			Func: w.runDeleteOpenBankingConnectionData,
		})
}

func (w Workflow) shouldContinueAsNew(ctx workflow.Context) bool {
	workflowInfo := workflow.GetInfo(ctx)
	return workflowInfo.GetContinueAsNewSuggested()
}
