package workflow

import (
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

type DeleteConnection struct {
	TaskID       models.TaskID
	ConnectorID  models.ConnectorID
	PsuID        uuid.UUID
	ConnectionID string
}

func (w Workflow) runDeleteConnection(
	ctx workflow.Context,
	deleteConnection DeleteConnection,
) error {
	if err := w.deleteConnection(ctx, deleteConnection); err != nil {
		errUpdateTask := w.updateTasksError(
			ctx,
			deleteConnection.TaskID,
			&deleteConnection.ConnectorID,
			err,
		)
		if errUpdateTask != nil {
			return errUpdateTask
		}

		return err
	}

	return w.updateTaskSuccess(
		ctx,
		deleteConnection.TaskID,
		&deleteConnection.ConnectorID,
		deleteConnection.PsuID.String(),
	)
}

func (w Workflow) deleteConnection(
	ctx workflow.Context,
	deletePSUConnection DeleteConnection,
) error {
	psu, err := activities.StoragePaymentServiceUsersGet(
		infiniteRetryContext(ctx),
		deletePSUConnection.PsuID,
	)
	if err != nil {
		return err
	}

	connection, _, err := activities.StorageOpenBankingConnectionsGetFromConnectionID(
		infiniteRetryContext(ctx),
		deletePSUConnection.ConnectorID,
		deletePSUConnection.ConnectionID,
	)
	if err != nil {
		return err
	}

	openBankingForwardedUser, err := activities.StorageOpenBankingForwardedUsersGet(
		infiniteRetryContext(ctx),
		deletePSUConnection.PsuID,
		deletePSUConnection.ConnectorID,
	)
	if err != nil {
		return err
	}

	if _, err := activities.PluginDeleteUserConnection(infiniteRetryContext(ctx), deletePSUConnection.ConnectorID, models.DeleteUserConnectionRequest{
		PaymentServiceUser:       models.ToPSPPaymentServiceUser(psu),
		OpenBankingForwardedUser: openBankingForwardedUser,
		Connection:               pointer.For(models.ToPSPOpenBankingConnection(*connection)),
	}); err != nil {
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
			FromConnectionID: &DeleteOpenBankingConnectionDataFromConnectionID{
				PSUID:        deletePSUConnection.PsuID,
				ConnectorID:  deletePSUConnection.ConnectorID,
				ConnectionID: deletePSUConnection.ConnectionID,
			},
		},
	).Get(ctx, nil); err != nil {
		return err
	}

	if err := activities.StorageOpenBankingConnectionsDelete(
		infiniteRetryContext(ctx),
		deletePSUConnection.PsuID,
		deletePSUConnection.ConnectorID,
		deletePSUConnection.ConnectionID,
	); err != nil {
		return err
	}

	return nil
}

const RunDeleteConnection = "DeleteConnection"
