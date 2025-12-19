package workflow

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func (s *UnitTestSuite) Test_Run_Periodically_FetchAccounts_Success() {
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, schedule models.Schedule) error {
		s.Equal(fmt.Sprintf("test-%s-FETCH_ACCOUNTS-1", s.connectorID.String()), schedule.ID)
		return nil
	})
	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.ScheduleCreateOptions) error {
		s.Equal(RunFetchNextAccounts, req.Action.Workflow)
		return nil
	})
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, schedule models.Schedule) error {
		s.Equal(fmt.Sprintf("test-%s-FETCH_PAYMENTS-1", s.connectorID.String()), schedule.ID)
		return nil
	})
	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.ScheduleCreateOptions) error {
		s.Equal(RunFetchNextPayments, req.Action.Workflow)
		return nil
	})

	s.env.ExecuteWorkflow(
		RunNextTasksV3_1,
		s.connectorID,
		&FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		[]models.ConnectorTaskTree{
			{
				TaskType:     models.TASK_FETCH_ACCOUNTS,
				Name:         "test",
				Periodically: true,
				NextTasks:    []models.ConnectorTaskTree{},
			},
			{
				TaskType:     models.TASK_FETCH_PAYMENTS,
				Name:         "test2",
				Periodically: true,
				NextTasks:    []models.ConnectorTaskTree{},
			},
		},
	)

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_Run_NoPeriodically_FetchAccounts_Success() {
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Return(func(ctx workflow.Context, req FetchNextAccounts, nextTasks []models.ConnectorTaskTree) error {
		s.Equal(s.connectorID, req.ConnectorID)
		s.False(req.Periodically)
		return nil
	})

	s.env.ExecuteWorkflow(
		RunNextTasksV3_1,
		s.connectorID,
		&FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		[]models.ConnectorTaskTree{
			{
				TaskType:     models.TASK_FETCH_ACCOUNTS,
				Name:         "test",
				Periodically: false,
				NextTasks:    []models.ConnectorTaskTree{},
			},
		},
	)

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_Run_Periodically_FetchNextExternalAccounts_Success() {
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, schedule models.Schedule) error {
		s.Equal(fmt.Sprintf("test-%s-FETCH_EXTERNAL_ACCOUNTS-1", s.connectorID.String()), schedule.ID)
		return nil
	})
	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.ScheduleCreateOptions) error {
		s.Equal(RunFetchNextExternalAccounts, req.Action.Workflow)
		return nil
	})

	s.env.ExecuteWorkflow(
		RunNextTasksV3_1,
		s.connectorID,
		&FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		[]models.ConnectorTaskTree{
			{
				TaskType:     models.TASK_FETCH_EXTERNAL_ACCOUNTS,
				Name:         "test",
				Periodically: true,
				NextTasks:    []models.ConnectorTaskTree{},
			},
		},
	)

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_Run_Periodically_FetchNextOthers_Success() {
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, schedule models.Schedule) error {
		s.Equal(fmt.Sprintf("test-%s-FETCH_OTHERS-1", s.connectorID.String()), schedule.ID)
		return nil
	})
	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.ScheduleCreateOptions) error {
		s.Equal(RunFetchNextOthers, req.Action.Workflow)
		return nil
	})

	s.env.ExecuteWorkflow(
		RunNextTasksV3_1,
		s.connectorID,
		&FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		[]models.ConnectorTaskTree{
			{
				TaskType:     models.TASK_FETCH_OTHERS,
				Name:         "test",
				Periodically: true,
				NextTasks:    []models.ConnectorTaskTree{},
			},
		},
	)

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_Run_Periodically_FetchNextPayments_Success() {
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, schedule models.Schedule) error {
		s.Equal(fmt.Sprintf("test-%s-FETCH_PAYMENTS-1", s.connectorID.String()), schedule.ID)
		return nil
	})
	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.ScheduleCreateOptions) error {
		s.Equal(RunFetchNextPayments, req.Action.Workflow)
		return nil
	})

	s.env.ExecuteWorkflow(
		RunNextTasksV3_1,
		s.connectorID,
		&FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		[]models.ConnectorTaskTree{
			{
				TaskType:     models.TASK_FETCH_PAYMENTS,
				Name:         "test",
				Periodically: true,
				NextTasks:    []models.ConnectorTaskTree{},
			},
		},
	)

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_Run_Periodically_FetchNextBalances_Success() {
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, schedule models.Schedule) error {
		s.Equal(fmt.Sprintf("test-%s-FETCH_BALANCES-1", s.connectorID.String()), schedule.ID)
		return nil
	})
	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.ScheduleCreateOptions) error {
		s.Equal(RunFetchNextBalances, req.Action.Workflow)
		return nil
	})

	s.env.ExecuteWorkflow(
		RunNextTasksV3_1,
		s.connectorID,
		&FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		[]models.ConnectorTaskTree{
			{
				TaskType:     models.TASK_FETCH_BALANCES,
				Name:         "test",
				Periodically: true,
				NextTasks:    []models.ConnectorTaskTree{},
			},
		},
	)

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_Run_Periodically_CreateWebhooks_Success() {
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, schedule models.Schedule) error {
		s.Equal(fmt.Sprintf("test-%s-CREATE_WEBHOOKS-1", s.connectorID.String()), schedule.ID)
		return nil
	})
	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.ScheduleCreateOptions) error {
		s.Equal(RunCreateWebhooks, req.Action.Workflow)
		return nil
	})

	s.env.ExecuteWorkflow(
		RunNextTasksV3_1,
		s.connectorID,
		&FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		[]models.ConnectorTaskTree{
			{
				TaskType:     models.TASK_CREATE_WEBHOOKS,
				Name:         "test",
				Periodically: true,
				NextTasks:    []models.ConnectorTaskTree{},
			},
		},
	)

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_Run_UnknownTaskType_Error() {
	s.env.ExecuteWorkflow(
		RunNextTasksV3_1,
		s.connectorID,
		&FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		[]models.ConnectorTaskTree{
			{
				TaskType:     100,
				Name:         "test",
				Periodically: true,
				NextTasks:    []models.ConnectorTaskTree{},
			},
		},
	)

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "unknown task type")
}

func (s *UnitTestSuite) Test_Run_StorageSchedulesStore_Error() {
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", fmt.Errorf("error-test")),
	)

	s.env.ExecuteWorkflow(
		RunNextTasksV3_1,
		s.connectorID,
		&FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		[]models.ConnectorTaskTree{
			{
				TaskType:     models.TASK_FETCH_ACCOUNTS,
				Name:         "test",
				Periodically: true,
				NextTasks:    []models.ConnectorTaskTree{},
			},
		},
	)

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_Run_TemporalScheduleCreate_Error() {
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", fmt.Errorf("error-test")),
	)

	s.env.ExecuteWorkflow(
		RunNextTasksV3_1,
		s.connectorID,
		&FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		[]models.ConnectorTaskTree{
			{
				TaskType:     models.TASK_FETCH_ACCOUNTS,
				Name:         "test",
				Periodically: true,
				NextTasks:    []models.ConnectorTaskTree{},
			},
		},
	)

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_Run_PreviousWorkflowVersion_Succeed() {
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Return(func(ctx workflow.Context, req FetchNextAccounts, nextTasks []models.ConnectorTaskTree) error {
		s.Equal(s.connectorID, req.ConnectorID)
		s.False(req.Periodically)
		return nil
	})

	s.env.ExecuteWorkflow(
		RunNextTasks, // nolint:staticcheck
		models.Config{},
		s.connectorID,
		&FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		[]models.ConnectorTaskTree{
			{
				TaskType:     models.TASK_FETCH_ACCOUNTS,
				Name:         "test",
				Periodically: false,
				NextTasks:    []models.ConnectorTaskTree{},
			},
		},
	)

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}
