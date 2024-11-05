package activities

import (
	"context"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) TemporalWorkflowExecutionsList(ctx context.Context, req *workflowservice.ListWorkflowExecutionsRequest) (*workflowservice.ListWorkflowExecutionsResponse, error) {
	return a.temporalClient.WorkflowService().ListWorkflowExecutions(ctx, req)
}

var TemporalWorkflowExecutionsListActivity = Activities{}.TemporalWorkflowExecutionsList

func TemporalWorkflowExecutionsList(ctx workflow.Context, req *workflowservice.ListWorkflowExecutionsRequest) (*workflowservice.ListWorkflowExecutionsResponse, error) {
	var resp workflowservice.ListWorkflowExecutionsResponse
	if err := executeActivity(ctx, TemporalWorkflowExecutionsListActivity, &resp, req); err != nil {
		return nil, err
	}
	return &resp, nil
}
