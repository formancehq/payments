package workflow

import (
	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

type DeletePSU struct {
	TaskID models.TaskID
	PsuID  uuid.UUID
}

func (w Workflow) runDeletePSU(
	ctx workflow.Context,
	deletePSU DeletePSU,
) error {
	if err := w.deletePSU(ctx, deletePSU); err != nil {
		errUpdateTask := w.updateTasksError(
			ctx,
			deletePSU.TaskID,
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
		deletePSU.TaskID,
		nil,
		deletePSU.PsuID.String(),
	)
}

func (w Workflow) deletePSU(
	ctx workflow.Context,
	deleteUser DeletePSU,
) error {
	psu, err := activities.StoragePaymentServiceUsersGet(infiniteRetryContext(ctx), deleteUser.PsuID)
	if err != nil {
		return err
	}

	// First, let's delete the user from all the open banking he is on.
	queryBB := storage.NewListOpenBankingForwardedUserQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.OpenBankingForwardedUserQuery{}).
			WithPageSize(100).
			WithQueryBuilder(
				query.Match("psu_id", deleteUser.PsuID.String()),
			),
	)
	for {
		forwardedUsers, err := activities.StorageOpenBankingForwardedUsersList(infiniteRetryContext(ctx), queryBB)
		if err != nil {
			return err
		}

		for _, forwardedUser := range forwardedUsers.Data {
			_, err = activities.PluginDeleteUser(
				infiniteRetryContext(ctx),
				forwardedUser.ConnectorID,
				models.DeleteUserRequest{
					PaymentServiceUser:       models.ToPSPPaymentServiceUser(psu),
					OpenBankingForwardedUser: &forwardedUser,
				},
			)
			if err != nil {
				return err
			}
		}

		if !forwardedUsers.HasMore {
			break
		}

		err = bunpaginate.UnmarshalCursor(forwardedUsers.Next, &queryBB)
		if err != nil {
			return err
		}
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
			PSUID: deleteUser.PsuID,
		},
	).Get(ctx, nil); err != nil {
		return err
	}

	if err := activities.StoragePaymentServiceUsersDelete(
		infiniteRetryContext(ctx),
		deleteUser.PsuID.String(),
	); err != nil {
		return err
	}

	return nil
}

const RunDeletePSU = "DeletePSU"
