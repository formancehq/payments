package workflow

import (
	"context"
	"errors"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
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

func (s *UnitTestSuite) Test_BootstrapTasks_PassesNextTasksForMatchingTaskType() {
	var capturedReq BootstrapTaskRequest
	s.env.OnWorkflow(RunBootstrapTask, mock.Anything, mock.Anything).Once().Return(
		func(ctx workflow.Context, req BootstrapTaskRequest) error {
			capturedReq = req
			return nil
		},
	)
	s.env.OnWorkflow(RunNextTasksV3_1, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	balancesSubtree := []models.ConnectorTaskTree{
		{TaskType: models.TASK_FETCH_BALANCES, Periodically: true},
	}

	s.env.ExecuteWorkflow(RunBootstrapTasks, BootstrapTasksRequest{
		ConnectorID: s.connectorID,
		TaskTypes:   []models.TaskType{models.TASK_FETCH_ACCOUNTS},
		TaskTree: []models.ConnectorTaskTree{
			{
				TaskType:  models.TASK_FETCH_ACCOUNTS,
				NextTasks: balancesSubtree,
			},
			// Unrelated top-level task must not leak its subtree into
			// the accounts bootstrap.
			{
				TaskType: models.TASK_FETCH_PAYMENTS,
				NextTasks: []models.ConnectorTaskTree{
					{TaskType: models.TASK_FETCH_CONVERSIONS},
				},
			},
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.Equal(models.TASK_FETCH_ACCOUNTS, capturedReq.TaskType)
	s.Equal(balancesSubtree, capturedReq.NextTasks)
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

func (s *UnitTestSuite) Test_BootstrapTasks_WithScheduleID_StoresAndTerminatesInstance() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(
		func(ctx context.Context, instance models.Instance) error {
			s.Equal("test-bootstrap-schedule", instance.ScheduleID)
			s.Equal(s.connectorID, instance.ConnectorID)
			s.False(instance.Terminated)
			return nil
		},
	)
	s.env.OnWorkflow(RunBootstrapTask, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunNextTasksV3_1, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(
		func(ctx context.Context, instance models.Instance) error {
			s.Equal("test-bootstrap-schedule", instance.ScheduleID)
			s.Equal(s.connectorID, instance.ConnectorID)
			s.True(instance.Terminated)
			s.NotNil(instance.TerminatedAt)
			s.Nil(instance.Error)
			return nil
		},
	)

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test-bootstrap-schedule")))
	s.NoError(err)

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

func (s *UnitTestSuite) Test_BootstrapTasks_WithScheduleID_OnFailure_PropagatesErrorToInstance() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunBootstrapTask, mock.Anything, mock.Anything).Once().Return(
		errors.New("bootstrap failed"),
	)
	// RunNextTasksV3_1 must NOT be called when bootstrap fails.
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(
		func(ctx context.Context, instance models.Instance) error {
			s.Equal("test-bootstrap-schedule", instance.ScheduleID)
			s.True(instance.Terminated)
			s.NotNil(instance.Error)
			s.Contains(*instance.Error, "bootstrap failed")
			return nil
		},
	)

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test-bootstrap-schedule")))
	s.NoError(err)

	s.env.ExecuteWorkflow(RunBootstrapTasks, BootstrapTasksRequest{
		ConnectorID: s.connectorID,
		TaskTypes:   []models.TaskType{models.TASK_FETCH_ACCOUNTS},
		TaskTree:    []models.ConnectorTaskTree{},
	})

	s.True(s.env.IsWorkflowCompleted())
	s.Error(s.env.GetWorkflowError())
}
