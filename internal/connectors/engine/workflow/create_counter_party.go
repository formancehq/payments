package workflow

import (
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type CreateCounterParty struct {
	TaskID         models.TaskID
	ConnectorID    models.ConnectorID
	CounterPartyID uuid.UUID
}

func (w Workflow) runCreateCounterParty(
	ctx workflow.Context,
	createCounterParty CreateCounterParty,
) error {
	accountID, err := w.createCounterParty(ctx, createCounterParty)
	if err != nil {
		if errUpdateTask := w.updateTasksError(
			ctx,
			createCounterParty.TaskID,
			&createCounterParty.ConnectorID,
			err,
		); errUpdateTask != nil {
			return errUpdateTask
		}

		return err
	}

	return w.updateTaskSuccess(
		ctx,
		createCounterParty.TaskID,
		&createCounterParty.ConnectorID,
		accountID,
	)
}

func (w Workflow) createCounterParty(
	ctx workflow.Context,
	createCounterParty CreateCounterParty,
) (string, error) {
	counterParty, err := activities.StorageCounterPartiesGet(
		infiniteRetryContext(ctx),
		createCounterParty.CounterPartyID,
	)
	if err != nil {
		return "", err
	}

	if counterParty.BankAccountID == nil {
		return "", temporal.NewNonRetryableApplicationError("bank account not found", "", errors.New("bank account not found"))
	}

	ba, err := activities.StorageBankAccountsGet(
		infiniteRetryContext(ctx),
		*counterParty.BankAccountID,
		true,
	)
	if err != nil {
		return "", err
	}

	createBAResponse, err := activities.PluginCreateBankAccount(
		infiniteRetryContext(ctx),
		createCounterParty.ConnectorID,
		models.CreateBankAccountRequest{
			CounterParty: pointer.For(models.ToPSPCounterParty(counterParty, ba)),
		},
	)
	if err != nil {
		return "", err
	}

	account := models.FromPSPAccount(
		createBAResponse.RelatedAccount,
		models.ACCOUNT_TYPE_EXTERNAL,
		createCounterParty.ConnectorID,
	)

	err = activities.StorageAccountsStore(
		infiniteRetryContext(ctx),
		[]models.Account{account},
	)
	if err != nil {
		return "", err
	}

	relatedAccount := models.CounterPartiesRelatedAccount{
		AccountID: account.ID,
		CreatedAt: createBAResponse.RelatedAccount.CreatedAt,
	}

	err = activities.StorageCounterPartiesAddRelatedAccount(
		infiniteRetryContext(ctx),
		createCounterParty.CounterPartyID,
		relatedAccount,
	)
	if err != nil {
		return "", err
	}

	counterParty.RelatedAccounts = append(counterParty.RelatedAccounts, relatedAccount)

	if err := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(
			ctx,
			workflow.ChildWorkflowOptions{
				TaskQueue:         w.getDefaultTaskQueue(),
				ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
				SearchAttributes: map[string]interface{}{
					SearchAttributeStack: w.stack,
				},
			},
		),
		RunSendEvents,
		SendEvents{
			CounterParty: counterParty,
		},
	).Get(ctx, nil); err != nil {
		return "", err
	}

	return account.ID.String(), nil
}

const RunCreateCounterParty = "CreateCounterParty"
