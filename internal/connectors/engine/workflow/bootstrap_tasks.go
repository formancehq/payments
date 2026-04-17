package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type BootstrapTasksRequest struct {
	ConnectorID models.ConnectorID         `json:"connectorID"`
	TaskTypes   []models.TaskType          `json:"taskTypes"`
	TaskTree    []models.ConnectorTaskTree `json:"taskTree"`
}

// runBootstrapTasks is a detached child of runInstallConnector. It runs each
// declared bootstrap task to completion in order, then — only on success —
// starts the exact same periodic scheduler the install workflow would have
// started if the plugin had not declared any bootstrap tasks. On failure,
// nothing is scheduled; operator recovery path is to re-install the
// connector, or to manually restart this workflow.
func (w Workflow) runBootstrapTasks(
	ctx workflow.Context,
	req BootstrapTasksRequest,
) error {
	for _, taskType := range req.TaskTypes {
		if err := workflow.ExecuteChildWorkflow(
			workflow.WithChildOptions(
				ctx,
				workflow.ChildWorkflowOptions{
					WorkflowID:        fmt.Sprintf("bootstrap-task-%s-%s-%d", w.stack, req.ConnectorID.String(), taskType),
					TaskQueue:         w.getDefaultTaskQueue(),
					ParentClosePolicy: enums.PARENT_CLOSE_POLICY_TERMINATE,
					SearchAttributes: map[string]interface{}{
						SearchAttributeStack: w.stack,
					},
				},
			),
			RunBootstrapTask,
			BootstrapTaskRequest{
				ConnectorID: req.ConnectorID,
				TaskType:    taskType,
			},
		).Get(ctx, nil); err != nil {
			return errors.Wrapf(err, "running bootstrap task %d", taskType)
		}
	}

	return w.startPeriodicSchedulesForBootstrap(ctx, req.ConnectorID, req.TaskTree)
}

// startPeriodicSchedulesForBootstrap launches the periodic scheduler child
// workflow, matching the WorkflowID / ParentClosePolicy / reuse policy that
// installConnector uses today. No version gate: this workflow is new, so
// there is no pre-existing history to replay — we always go through the
// current `RunNextTasksV3_1` entry point.
func (w Workflow) startPeriodicSchedulesForBootstrap(
	ctx workflow.Context,
	connectorID models.ConnectorID,
	taskTree []models.ConnectorTaskTree,
) error {
	childOpts := workflow.ChildWorkflowOptions{
		WorkflowID:            fmt.Sprintf("run-tasks-%s-%s", w.stack, connectorID.String()),
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
		TaskQueue:             w.getDefaultTaskQueue(),
		ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
		SearchAttributes: map[string]interface{}{
			SearchAttributeStack: w.stack,
		},
	}

	if err := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(ctx, childOpts),
		RunNextTasksV3_1,
		connectorID,
		nil,
		taskTree,
	).GetChildWorkflowExecution().Get(ctx, nil); err != nil {
		if temporal.IsWorkflowExecutionAlreadyStartedError(err) {
			return nil
		}
		return errors.Wrap(err, "running next workflow after bootstrap")
	}
	return nil
}

const RunBootstrapTasks = "RunBootstrapTasks"
