package workflow

import (
	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type DeleteUser struct {
	TaskID models.TaskID
	PsuID  uuid.UUID
}

func (w Workflow) runDeleteUser(
	ctx workflow.Context,
	deleteUser DeleteUser,
) error {
	if err := w.deleteUser(ctx, deleteUser); err != nil {
		errUpdateTask := w.updateTasksError(
			ctx,
			deleteUser.TaskID,
			nil,
			err,
		)
		if errUpdateTask != nil {
			return errUpdateTask
		}

		return err
	}

	return w.updateTaskSuccess(
		ctx,
		deleteUser.TaskID,
		nil,
		deleteUser.PsuID.String(),
	)
}

func (w Workflow) deleteUser(
	ctx workflow.Context,
	deleteUser DeleteUser,
) error {
	psu, err := activities.StoragePaymentServiceUsersGet(infiniteRetryContext(ctx), deleteUser.PsuID)
	if err != nil {
		return err
	}

	query := storage.NewListPSUBankBridgesQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.PSUBankBridgesQuery{}).
			WithPageSize(100).
			WithQueryBuilder(
				query.Match("psu_id", deleteUser.PsuID.String()),
			),
	)
	for {
		psuBankBridges, err := activities.StoragePSUBankBridgesList(infiniteRetryContext(ctx), query)
		if err != nil {
			return err
		}

		for _, psuBankBridge := range psuBankBridges.Data {
			_, err = activities.PluginDeleteUser(
				infiniteRetryContext(ctx),
				psuBankBridge.ConnectorID,
				models.DeleteUserRequest{
					PaymentServiceUser: models.ToPSPPaymentServiceUser(psu),
					PSUBankBridge:      &psuBankBridge,
				},
			)
			if err != nil {
				return err
			}
		}

		if !psuBankBridges.HasMore {
			break
		}

		err = bunpaginate.UnmarshalCursor(psuBankBridges.Next, &query)
		if err != nil {
			return err
		}
	}

	return nil
}

const RunDeleteUser = "DeleteUser"
