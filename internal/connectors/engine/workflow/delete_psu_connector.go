package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

type DeletePSUConnector struct {
	TaskID      models.TaskID
	PsuID       uuid.UUID
	ConnectorID models.ConnectorID
}

func (w Workflow) runDeletePSUConnector(
	ctx workflow.Context,
	deletePSUConnector DeletePSUConnector,
) error {
	if err := w.deletePSUConnector(ctx, deletePSUConnector); err != nil {
		errUpdateTask := w.updateTasksError(
			ctx,
			deletePSUConnector.TaskID,
			&deletePSUConnector.ConnectorID,
			err,
		)
		if errUpdateTask != nil {
			return errUpdateTask
		}

		return err
	}

	return w.updateTaskSuccess(
		ctx,
		deletePSUConnector.TaskID,
		&deletePSUConnector.ConnectorID,
		deletePSUConnector.PsuID.String(),
	)
}

func (w Workflow) deletePSUConnector(
	ctx workflow.Context,
	deletePSUConnector DeletePSUConnector,
) error {
	psu, err := activities.StoragePaymentServiceUsersGet(
		infiniteRetryContext(ctx),
		deletePSUConnector.PsuID,
	)
	if err != nil {
		return err
	}

	openBankingForwardedUser, err := activities.StorageOpenBankingForwardedUsersGet(
		infiniteRetryContext(ctx),
		deletePSUConnector.PsuID,
		deletePSUConnector.ConnectorID,
	)
	if err != nil {
		return err
	}

	_, err = activities.PluginDeleteUser(
		infiniteRetryContext(ctx),
		deletePSUConnector.ConnectorID,
		models.DeleteUserRequest{
			PaymentServiceUser:       models.ToPSPPaymentServiceUser(psu),
			OpenBankingForwardedUser: openBankingForwardedUser,
		},
	)
	if err != nil {
		return err
	}

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
		RunDeleteOpenBankingConnectionData,
		DeleteOpenBankingConnectionData{
			FromConnectorID: &DeleteOpenBankingConnectionDataFromConnectorID{
				PSUID:       deletePSUConnector.PsuID,
				ConnectorID: deletePSUConnector.ConnectorID,
			},
		},
	).Get(ctx, nil); err != nil {
		return err
	}

	if err := activities.StorageOpenBankingForwardedUsersDelete(
		infiniteRetryContext(ctx),
		deletePSUConnector.PsuID,
		deletePSUConnector.ConnectorID,
	); err != nil {
		return err
	}

	return nil
}

var RunDeletePSUConnector = "DeletePSUConnector"
