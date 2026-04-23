package workflow

import (
	"errors"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/workflow"
)

func (s *UnitTestSuite) Test_BootstrapTask_DelegatesToRunFetchNextAccounts() {
	var capturedReq FetchNextAccounts
	var capturedNextTasks []models.ConnectorTaskTree

	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		func(ctx workflow.Context, req FetchNextAccounts, nextTasks []models.ConnectorTaskTree) error {
			capturedReq = req
			capturedNextTasks = nextTasks
			return nil
		},
	)

	nextTasks := []models.ConnectorTaskTree{
		{TaskType: models.TASK_FETCH_BALANCES, Periodically: true},
	}

	s.env.ExecuteWorkflow(RunBootstrapTask, BootstrapTaskRequest{
		ConnectorID: s.connectorID,
		TaskType:    models.TASK_FETCH_ACCOUNTS,
		NextTasks:   nextTasks,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.Equal(s.connectorID, capturedReq.ConnectorID)
	s.Nil(capturedReq.FromPayload, "bootstrap state key must match periodic (FromPayload=nil)")
	s.False(capturedReq.Periodically, "bootstrap is not a periodic run")
	s.Equal(nextTasks, capturedNextTasks, "bootstrap must forward per-account fan-out subtree")
}

func (s *UnitTestSuite) Test_BootstrapTask_UnrecognizedTaskType_NonRetryableError() {
	s.env.ExecuteWorkflow(RunBootstrapTask, BootstrapTaskRequest{
		ConnectorID: s.connectorID,
		TaskType:    models.TaskType(9999),
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.Contains(err.Error(), "bootstrap does not support task type")
}

func (s *UnitTestSuite) Test_BootstrapTask_FetchAccountsFailure_Propagates() {
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		errors.New("upstream API failure"),
	)

	s.env.ExecuteWorkflow(RunBootstrapTask, BootstrapTaskRequest{
		ConnectorID: s.connectorID,
		TaskType:    models.TASK_FETCH_ACCOUNTS,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.Contains(err.Error(), "bootstrap fetching accounts")
}
