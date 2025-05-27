package workflow

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_TerminateWorkflows_Success() {
	s.env.OnActivity(activities.TemporalWorkflowExecutionsListActivity, mock.Anything, mock.Anything).Once().Return(
		&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				{
					Execution: &common.WorkflowExecution{
						WorkflowId: "test-workflow",
						RunId:      "test-run",
					},
					Status: enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
				},
			},
		},
		nil,
	)
	s.env.OnActivity(activities.TemporalWorkflowTerminateActivity, mock.Anything, "test-workflow", "test-run", "uninstalling connector").Once().Return(nil)

	s.env.ExecuteWorkflow(RunTerminateWorkflows, TerminateWorkflows{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_TerminateWorkflows_EmptyWorkflows_Success() {
	s.env.OnActivity(activities.TemporalWorkflowExecutionsListActivity, mock.Anything, mock.Anything).Once().Return(
		&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{},
		},
		nil,
	)

	s.env.ExecuteWorkflow(RunTerminateWorkflows, TerminateWorkflows{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_TerminateWorkflows_NotRunningWorkflows_Success() {
	s.env.OnActivity(activities.TemporalWorkflowExecutionsListActivity, mock.Anything, mock.Anything).Once().Return(
		&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				{
					Execution: &common.WorkflowExecution{
						WorkflowId: "test-workflow",
						RunId:      "test-run",
					},
					Status: enums.WORKFLOW_EXECUTION_STATUS_CANCELED,
				},
			},
		},
		nil,
	)

	s.env.ExecuteWorkflow(RunTerminateWorkflows, TerminateWorkflows{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_TerminateWorkflows_TemporalWorkflowExecutionsList_Error() {
	s.env.OnActivity(activities.TemporalWorkflowExecutionsListActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunTerminateWorkflows, TerminateWorkflows{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_TerminateWorkflows_TemporalWorkflowTerminate_Error() {
	s.env.OnActivity(activities.TemporalWorkflowExecutionsListActivity, mock.Anything, mock.Anything).Once().Return(
		&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				{
					Execution: &common.WorkflowExecution{
						WorkflowId: "test-workflow",
						RunId:      "test-run",
					},
					Status: enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
				},
			},
		},
		nil,
	)
	s.env.OnActivity(activities.TemporalWorkflowTerminateActivity, mock.Anything, "test-workflow", "test-run", "uninstalling connector").Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunTerminateWorkflows, TerminateWorkflows{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}
