package workflow

import (
	"context"
	"fmt"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/workflow"
)

func (w Workflow) runTerminateWorkflows(
	ctx workflow.Context,
	uninstallConnector UninstallConnector,
) error {
	var nextPageToken []byte

	for {
		resp, err := w.temporalClient.WorkflowService().ListWorkflowExecutions(
			context.Background(),
			&workflowservice.ListWorkflowExecutionsRequest{
				Namespace:     w.temporalNamespace,
				PageSize:      100,
				NextPageToken: nextPageToken,
				Query:         fmt.Sprintf("Stack=\"%s\" and TaskQueue=\"%s\"", w.stack, uninstallConnector.ConnectorID.String()),
			},
		)
		if err != nil {
			return err
		}

		wg := workflow.NewWaitGroup(ctx)
		errChan := make(chan error, len(resp.Executions))
		for _, e := range resp.Executions {
			if e.Status != enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
				continue
			}

			wg.Add(1)
			workflow.Go(ctx, func(ctx workflow.Context) {
				defer wg.Done()
				if err := w.temporalClient.TerminateWorkflow(
					context.Background(),
					e.Execution.WorkflowId,
					e.Execution.RunId,
					"uninstalling connector",
				); err != nil {
					switch err.(type) {
					case *serviceerror.NotFound:
						// Do nothing, the workflow is already terminated
						return
					default:
						errChan <- err
					}
				}
			})
		}

		wg.Wait(ctx)
		close(errChan)

		for err := range errChan {
			if err != nil {
				return err
			}
		}

		if resp.NextPageToken == nil {
			break
		}

		nextPageToken = resp.NextPageToken
	}

	return nil
}

var RunTerminateWorkflows any

func init() {
	RunTerminateWorkflows = Workflow{}.runTerminateWorkflows
}
