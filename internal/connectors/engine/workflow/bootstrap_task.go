package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type BootstrapTaskRequest struct {
	ConnectorID models.ConnectorID         `json:"connectorID"`
	TaskType    models.TaskType            `json:"taskType"`
	NextTasks   []models.ConnectorTaskTree `json:"nextTasks"`
}

// runBootstrapTask delegates each bootstrap task to the existing periodic
// fetch workflow, run synchronously as a child workflow. This reuses the
// full pagination loop — per-page state persistence, per-account fan-out,
// ContinueAsNew on large histories, and connector_instances tracking —
// rather than maintaining a parallel implementation.
//
// Bootstrap semantics vs. periodic:
// - `Periodically: false` so no new schedule is created from within
//   the loop (RunBootstrapTasks starts the periodic scheduler itself,
//   post-bootstrap); plugins that read the flag can distinguish the
//   install-time pass from a scheduled tick.
// - `FromPayload: nil` so the state key matches the periodic scheduler's
//   (`CAPABILITY_FETCH_ACCOUNTS`) and the next periodic tick picks up
//   from the cursor this run stored.
func (w Workflow) runBootstrapTask(
	ctx workflow.Context,
	req BootstrapTaskRequest,
) error {
	switch req.TaskType {
	case models.TASK_FETCH_ACCOUNTS:
		if err := workflow.ExecuteChildWorkflow(
			workflow.WithChildOptions(
				ctx,
				workflow.ChildWorkflowOptions{
					WorkflowID:        fmt.Sprintf("bootstrap-fetch-accounts-%s-%s", w.stack, req.ConnectorID.String()),
					TaskQueue:         w.getDefaultTaskQueue(),
					ParentClosePolicy: enums.PARENT_CLOSE_POLICY_TERMINATE,
					SearchAttributes: map[string]interface{}{
						SearchAttributeStack: w.stack,
					},
				},
			),
			RunFetchNextAccounts,
			FetchNextAccounts{
				ConnectorID:  req.ConnectorID,
				FromPayload:  nil,
				Periodically: false,
			},
			req.NextTasks,
		).Get(ctx, nil); err != nil {
			return errors.Wrap(err, "bootstrap fetching accounts")
		}
		return nil
	default:
		return temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("bootstrap does not support task type %d", req.TaskType),
			ErrValidation,
			nil,
		)
	}
}

const RunBootstrapTask = "RunBootstrapTask"
