package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type CreateBankAccount struct {
	TaskID        models.TaskID
	ConnectorID   models.ConnectorID
	BankAccountID uuid.UUID
}

func (w Workflow) runCreateBankAccount(
	ctx workflow.Context,
	createBankAccount CreateBankAccount,
) error {
	accountID, err := w.createBankAccount(ctx, createBankAccount)
	if err != nil {
		if errUpdateTask := w.updateTasksError(
			ctx,
			createBankAccount.TaskID,
			&createBankAccount.ConnectorID,
			err,
		); errUpdateTask != nil {
			return errUpdateTask
		}

		return err
	}

	return w.updateTaskSuccess(
		ctx,
		createBankAccount.TaskID,
		&createBankAccount.ConnectorID,
		accountID,
	)
}

func (w Workflow) createBankAccount(
	ctx workflow.Context,
	createBankAccount CreateBankAccount,
) (string, error) {
	bankAccount, err := activities.StorageBankAccountsGet(
		infiniteRetryContext(ctx),
		createBankAccount.BankAccountID,
		true,
	)
	if err != nil {
		return "", err
	}

	createBAResponse, err := activities.PluginCreateBankAccount(
		infiniteRetryContext(ctx),
		createBankAccount.ConnectorID,
		models.CreateBankAccountRequest{
			BankAccount: *bankAccount,
		},
	)
	if err != nil {
		return "", err
	}

	account, err := models.FromPSPAccount(
		createBAResponse.RelatedAccount,
		models.ACCOUNT_TYPE_EXTERNAL,
		createBankAccount.ConnectorID,
	)
	if err != nil {
		return "", temporal.NewNonRetryableApplicationError(
			"failed to translate accounts",
			ErrValidation,
			err,
		)
	}

	err = activities.StorageAccountsStore(
		infiniteRetryContext(ctx),
		[]models.Account{account},
	)
	if err != nil {
		return "", err
	}

	relatedAccount := models.BankAccountRelatedAccount{
		AccountID: account.ID,
		CreatedAt: createBAResponse.RelatedAccount.CreatedAt,
	}

	err = activities.StorageBankAccountsAddRelatedAccount(
		infiniteRetryContext(ctx),
		createBankAccount.BankAccountID,
		relatedAccount,
	)
	if err != nil {
		return "", err
	}

	bankAccount.RelatedAccounts = append(bankAccount.RelatedAccounts, relatedAccount)

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
			BankAccount: bankAccount,
		},
	).Get(ctx, nil); err != nil {
		return "", err
	}

	return account.ID.String(), nil
}

const RunCreateBankAccount = "CreateBankAccount"
