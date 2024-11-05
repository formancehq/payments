package activities

import (
	"context"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) TemporalWorkflowTerminate(ctx context.Context, workflowID string, runID string, reason string) error {
	err := a.temporalClient.TerminateWorkflow(
		ctx,
		workflowID,
		runID,
		reason,
	)
	if err != nil {
		switch err.(type) {
		case *serviceerror.NotFound:
			// Do nothing, the workflow is already terminated
			return nil
		default:
			return err
		}
	}
	return nil
}

var TemporalWorkflowTerminateActivity = Activities{}.TemporalWorkflowTerminate

func TemporalWorkflowTerminate(ctx workflow.Context, workflowID string, runID string, reason string) error {
	return executeActivity(ctx, TemporalWorkflowTerminateActivity, nil, workflowID, runID, reason)
}
