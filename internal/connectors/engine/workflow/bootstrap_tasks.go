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
	if err := w.createInstance(ctx, req.ConnectorID); err != nil {
		return errors.Wrap(err, "creating instance for "+RunBootstrapTasks)
	}
	err := w.bootstrapTasks(ctx, req)
	return w.terminateInstance(ctx, req.ConnectorID, err)
}

func (w Workflow) bootstrapTasks(
	ctx workflow.Context,
	req BootstrapTasksRequest,
) error {
	for _, taskType := range req.TaskTypes {
		// Pass the per-task subtree (the NextTasks under the matching
		// top-level entry in the connector's ConnectorTasksTree) so the
		// bootstrap loop can fan out the same per-record downstream tasks
		// the periodic workflow would have created. Without this, plugins
		// that do incremental fetch on the periodic tick would never
		// register the per-account schedules for bootstrapped accounts.
		nextTasks := findNextTasksForType(req.TaskTree, taskType)

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
				NextTasks:   nextTasks,
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

// bootstrapScheduleID returns the deterministic schedule ID for the one-shot
// bootstrap schedule associated with a connector. Single source of truth for
// the format so install/uninstall/cleanup/tests cannot drift.
func (w Workflow) bootstrapScheduleID(connectorID models.ConnectorID) string {
	return fmt.Sprintf("bootstrap-%s-%s", w.stack, connectorID.String())
}

// findNextTasksForType returns the NextTasks subtree for the top-level
// entry in taskTree whose TaskType matches. Returns nil if no top-level
// entry matches — in which case the per-task bootstrap runs without any
// fan-out, matching the plugin's declared task tree.
func findNextTasksForType(taskTree []models.ConnectorTaskTree, taskType models.TaskType) []models.ConnectorTaskTree {
	for _, task := range taskTree {
		if task.TaskType == taskType {
			return task.NextTasks
		}
	}
	return nil
}

const RunBootstrapTasks = "RunBootstrapTasks"
