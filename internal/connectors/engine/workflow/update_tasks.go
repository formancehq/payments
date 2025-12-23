package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"go.temporal.io/api/enums/v1"
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

	// Task events are now sent via outbox pattern in TasksUpsert
	// (unless it's a rerun from a previous version, in which case:)
	if !IsEventOutboxPatternEnabled(ctx) {
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
			RunSendEvents, //nolint:staticcheck // ignore deprecation
			SendEvents{
				Task: &task,
			},
		).Get(ctx, nil); err != nil {
			return err
		}
	}
	return nil
}
