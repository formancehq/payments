package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

type CreateFormanceAccount struct {
	Account models.Account
}

func (w Workflow) runCreateFormanceAccount(
	ctx workflow.Context,
	createFormanceAccount CreateFormanceAccount,
) error {
	err := activities.StorageAccountsStore(
		infiniteRetryContext(ctx),
		[]models.Account{createFormanceAccount.Account},
	)
	if err != nil {
		return err
	}

	if err := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(
			ctx,
			workflow.ChildWorkflowOptions{
				TaskQueue:         createFormanceAccount.Account.ConnectorID.String(),
				ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
				SearchAttributes: map[string]interface{}{
					SearchAttributeStack: w.stack,
				},
			},
		),
		RunSendEvents,
		SendEvents{
			Account: &createFormanceAccount.Account,
		},
	).Get(ctx, nil); err != nil {
		return err
	}

	return nil
}

var RunCreateFormanceAccount any

func init() {
	RunCreateFormanceAccount = Workflow{}.runCreateFormanceAccount
}
