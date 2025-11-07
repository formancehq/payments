package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"go.temporal.io/sdk/workflow"
)

func (w Workflow) updateTasksError(
	ctx workflow.Context,
	taskID models.TaskID,
	connectorID *models.ConnectorID,
	cause error,
) error {
	cause = errorsutils.Cause(cause)
	task := models.Task{
		ID:          taskID,
		ConnectorID: connectorID,
		Status:      models.TASK_STATUS_FAILED,
		UpdatedAt:   workflow.Now(ctx).UTC(),
		Error:       cause,
	}

	return w.updateTask(ctx, task)
}

func (w Workflow) updateTaskSuccess(
	ctx workflow.Context,
	taskID models.TaskID,
	connectorID *models.ConnectorID,
	relatedObjectID string,
) error {
	task := models.Task{
		ID:              taskID,
		ConnectorID:     connectorID,
		Status:          models.TASK_STATUS_SUCCEEDED,
		UpdatedAt:       workflow.Now(ctx).UTC(),
		CreatedObjectID: &relatedObjectID,
	}

	return w.updateTask(ctx, task)
}

func (w Workflow) updateTask(ctx workflow.Context, task models.Task) error {
	if err := activities.StorageTasksStore(
		infiniteRetryContext(ctx),
		task,
	); err != nil {
		return err
	}

	if err := w.runSendEvents(ctx, SendEvents{
		Task: &task,
	}); err != nil {
		return fmt.Errorf("sending events: %w", err)
	}

	return nil
}
