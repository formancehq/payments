package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"go.temporal.io/sdk/workflow"
)

func (w Workflow) updateTasksError(
	ctx workflow.Context,
	taskID models.TaskID,
	connectorID *models.ConnectorID,
	err error,
) error {
	cause := errorsutils.Cause(err)
	return activities.StorageTasksStore(
		infiniteRetryContext(ctx),
		models.Task{
			ID:          taskID,
			ConnectorID: connectorID,
			Status:      models.TASK_STATUS_FAILED,
			UpdatedAt:   workflow.Now(ctx).UTC(),
			Error:       cause,
		})
}

func (w Workflow) updateTaskSuccess(
	ctx workflow.Context,
	taskID models.TaskID,
	connectorID *models.ConnectorID,
	relatedObjectID string,
) error {
	return activities.StorageTasksStore(
		infiniteRetryContext(ctx),
		models.Task{
			ID:              taskID,
			ConnectorID:     connectorID,
			Status:          models.TASK_STATUS_SUCCEEDED,
			UpdatedAt:       workflow.Now(ctx).UTC(),
			CreatedObjectID: &relatedObjectID,
		})
}
