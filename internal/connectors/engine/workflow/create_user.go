package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type CreateUser struct {
	TaskID      models.TaskID
	ConnectorID models.ConnectorID
	PsuID       uuid.UUID
}

func (w Workflow) runCreateUser(
	ctx workflow.Context,
	createUser CreateUser,
) error {
	err := w.createUser(ctx, createUser)
	if err != nil {
		errUpdateTask := w.updateTasksError(
			ctx,
			createUser.TaskID,
			&createUser.ConnectorID,
			err,
		)
		if errUpdateTask != nil {
			return errUpdateTask
		}

		return err
	}

	return w.updateTaskSuccess(
		ctx,
		createUser.TaskID,
		&createUser.ConnectorID,
		createUser.PsuID.String(),
	)
}

func (w Workflow) createUser(
	ctx workflow.Context,
	createUser CreateUser,
) error {
	psu, err := activities.StoragePaymentServiceUsersGet(
		infiniteRetryContext(ctx),
		createUser.PsuID,
	)
	if err != nil {
		return err
	}

	resp, err := activities.PluginCreateUser(
		infiniteRetryContext(ctx),
		createUser.ConnectorID,
		models.CreateUserRequest{
			PaymentServiceUser: models.ToPSPPaymentServiceUser(psu),
		},
	)
	if err != nil {
		return err
	}

	bankBridge := models.PSUBankBridge{
		ConnectorID: createUser.ConnectorID,
		AccessToken: resp.PermanentToken,
		Metadata:    resp.Metadata,
	}

	err = activities.StoragePSUBankBridgesStore(
		infiniteRetryContext(ctx),
		createUser.PsuID,
		bankBridge,
	)
	if err != nil {
		return err
	}

	return nil
}

const RunCreateUser = "CreateUser"
