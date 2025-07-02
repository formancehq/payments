package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type DeleteUserConnection struct {
	TaskID       models.TaskID
	ConnectorID  models.ConnectorID
	PsuID        uuid.UUID
	ConnectionID string
}

func (w Workflow) runDeleteUserConnection(
	ctx workflow.Context,
	deleteUserConnection DeleteUserConnection,
) error {
	psu, err := activities.StoragePaymentServiceUsersGet(
		ctx,
		deleteUserConnection.PsuID,
	)
	if err != nil {
		return err
	}

	psuBankBridge, err := activities.StoragePSUBankBridgesGet(
		ctx,
		deleteUserConnection.PsuID,
		deleteUserConnection.ConnectorID,
	)
	if err != nil {
		return err
	}

	activities.PluginDeleteUserConnection(ctx, deleteUserConnection.ConnectorID, models.DeleteUserConnectionRequest{
		PaymentServiceUser: models.ToPSPPaymentServiceUser(psu),
		PSUBankBridge:      psuBankBridge,
		Connection:         &models.PSUBankBridgeConnection{},
	})
	return nil
}

const RunDeleteUserConnection = "DeleteUserConnection"
