package workflow

import (
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

type DeletePSUConnection struct {
	TaskID       models.TaskID
	ConnectorID  models.ConnectorID
	PsuID        uuid.UUID
	ConnectionID string
}

func (w Workflow) runDeletePSUConnection(
	ctx workflow.Context,
	deletePSUConnection DeletePSUConnection,
) error {
	if err := w.deletePSUConnection(ctx, deletePSUConnection); err != nil {
		errUpdateTask := w.updateTasksError(
			ctx,
			deletePSUConnection.TaskID,
			&deletePSUConnection.ConnectorID,
			err,
		)
		if errUpdateTask != nil {
			return errUpdateTask
		}

		return err
	}

	return w.updateTaskSuccess(
		ctx,
		deletePSUConnection.TaskID,
		&deletePSUConnection.ConnectorID,
		deletePSUConnection.PsuID.String(),
	)
}

func (w Workflow) deletePSUConnection(
	ctx workflow.Context,
	deletePSUConnection DeletePSUConnection,
) error {
	psu, err := activities.StoragePaymentServiceUsersGet(
		infiniteRetryContext(ctx),
		deletePSUConnection.PsuID,
	)
	if err != nil {
		return err
	}

	connection, _, err := activities.StoragePSUBankBridgeConnectionsGetFromConnectionID(
		infiniteRetryContext(ctx),
		deletePSUConnection.ConnectorID,
		deletePSUConnection.ConnectionID,
	)
	if err != nil {
		return err
	}

	psuBankBridge, err := activities.StoragePSUBankBridgesGet(
		infiniteRetryContext(ctx),
		deletePSUConnection.PsuID,
		deletePSUConnection.ConnectorID,
	)
	if err != nil {
		return err
	}

	if _, err := activities.PluginDeleteUserConnection(infiniteRetryContext(ctx), deletePSUConnection.ConnectorID, models.DeleteUserConnectionRequest{
		PaymentServiceUser: models.ToPSPPaymentServiceUser(psu),
		PSUBankBridge:      psuBankBridge,
		Connection:         pointer.For(models.ToPSPPsuBankBridgeConnection(*connection)),
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
		RunDeleteBankBridgeConnectionData,
		DeleteBankBridgeConnectionData{
			PSUID: deletePSUConnection.PsuID,
			FromConnectionID: &DeleteBankBridgeConnectionDataFromConnectionID{
				ConnectionID: deletePSUConnection.ConnectionID,
			},
		},
	).Get(ctx, nil); err != nil {
		return err
	}

	if err := activities.StoragePSUBankBridgeConnectionDelete(
		infiniteRetryContext(ctx),
		deletePSUConnection.PsuID,
		deletePSUConnection.ConnectorID,
		deletePSUConnection.ConnectionID,
	); err != nil {
		return err
	}

	return nil
}

const RunDeletePSUConnection = "DeletePSUConnection"
