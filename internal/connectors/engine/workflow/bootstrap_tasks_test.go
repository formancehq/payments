package workflow

import (
	"errors"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/workflow"
)

func (s *UnitTestSuite) Test_BootstrapTasks_Success_StartsPeriodicSchedule() {
	s.env.OnWorkflow(RunBootstrapTask, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunNextTasksV3_1, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunBootstrapTasks, BootstrapTasksRequest{
		ConnectorID: s.connectorID,
		TaskTypes:   []models.TaskType{models.TASK_FETCH_ACCOUNTS},
		TaskTree: []models.ConnectorTaskTree{
			{TaskType: models.TASK_FETCH_ACCOUNTS},
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_BootstrapTasks_MultipleTasks_RunSequentially() {
	call := 0
	s.env.OnWorkflow(RunBootstrapTask, mock.Anything, mock.Anything).Times(2).Return(
		func(ctx workflow.Context, req BootstrapTaskRequest) error {
			call++
			return nil
		},
	)
	s.env.OnWorkflow(RunNextTasksV3_1, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunBootstrapTasks, BootstrapTasksRequest{
		ConnectorID: s.connectorID,
		TaskTypes:   []models.TaskType{models.TASK_FETCH_ACCOUNTS, models.TASK_FETCH_OTHERS},
		TaskTree:    []models.ConnectorTaskTree{},
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.Equal(2, call)
}

func (s *UnitTestSuite) Test_BootstrapTasks_BootstrapFailure_DoesNotSchedulePeriodic() {
	s.env.OnWorkflow(RunBootstrapTask, mock.Anything, mock.Anything).Once().Return(
		errors.New("bootstrap failed"),
	)
	// RunNextTasksV3_1 must NOT be called when bootstrap fails.

	s.env.ExecuteWorkflow(RunBootstrapTasks, BootstrapTasksRequest{
		ConnectorID: s.connectorID,
		TaskTypes:   []models.TaskType{models.TASK_FETCH_ACCOUNTS},
		TaskTree:    []models.ConnectorTaskTree{},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_BootstrapTasks_EmptyTaskList_StartsPeriodicScheduleDirectly() {
	// Declaring BootstrapOnInstall returns an empty slice is a no-op:
	// RunBootstrapTasks should go straight to starting the periodic scheduler.
	s.env.OnWorkflow(RunNextTasksV3_1, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunBootstrapTasks, BootstrapTasksRequest{
		ConnectorID: s.connectorID,
		TaskTypes:   []models.TaskType{},
		TaskTree:    []models.ConnectorTaskTree{},
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}
