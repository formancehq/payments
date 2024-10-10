package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

type CreateFormancePayment struct {
	Payment models.Payment
}

func (w Workflow) runCreateFormancePayment(
	ctx workflow.Context,
	createFormancePayment CreateFormancePayment,
) error {
	err := activities.StoragePaymentsStore(
		infiniteRetryContext(ctx),
		[]models.Payment{createFormancePayment.Payment},
	)
	if err != nil {
		return err
	}

	if err := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(
			ctx,
			workflow.ChildWorkflowOptions{
				TaskQueue:         createFormancePayment.Payment.ConnectorID.String(),
				ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
				SearchAttributes: map[string]interface{}{
					SearchAttributeStack: w.stack,
				},
			},
		),
		RunSendEvents,
		SendEvents{
			Payment: &createFormancePayment.Payment,
		},
	).Get(ctx, nil); err != nil {
		return err
	}

	return nil
}

var RunCreateFormancePayment any

func init() {
	RunCreateFormancePayment = Workflow{}.runCreateFormancePayment
}
