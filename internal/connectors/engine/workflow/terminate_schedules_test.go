package workflow

import (
	"errors"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_TerminateSchedules_Success() {
	s.env.OnActivity(activities.StorageSchedulesListActivity, mock.Anything, mock.Anything).Once().Return(
		&bunpaginate.Cursor[models.Schedule]{
			PageSize: 1,
			HasMore:  false,
			Data: []models.Schedule{
				{
					ID:          "test",
					ConnectorID: s.connectorID,
					CreatedAt:   s.env.Now(),
				},
			},
		},
		nil,
	)
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, "test").Once().Return(nil)

	s.env.ExecuteWorkflow(RunTerminateSchedules, UninstallConnector{
		ConnectorID:       s.connectorID,
		DefaultWorkerName: "test",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_TerminateSchedules_EmptyScheduleList_Success() {
	s.env.OnActivity(activities.StorageSchedulesListActivity, mock.Anything, mock.Anything).Once().Return(
		&bunpaginate.Cursor[models.Schedule]{
			PageSize: 0,
			HasMore:  false,
			Data:     []models.Schedule{},
		},
		nil,
	)

	s.env.ExecuteWorkflow(RunTerminateSchedules, UninstallConnector{
		ConnectorID:       s.connectorID,
		DefaultWorkerName: "test",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_TerminateSchedules_HasMore_Success() {
	s.env.OnActivity(activities.StorageSchedulesListActivity, mock.Anything, mock.Anything).Once().Return(
		&bunpaginate.Cursor[models.Schedule]{
			PageSize: 1,
			HasMore:  true,
			Next:     "eyJvZmZzZXQiOjYsIm9yZGVyIjowLCJwYWdlU2l6ZSI6MywiZmlsdGVycyI6eyJxYiI6bnVsbCwicGFnZVNpemUiOjMsIm9wdGlvbnMiOnt9fX0",
			Data: []models.Schedule{
				{
					ID:          "test",
					ConnectorID: s.connectorID,
					CreatedAt:   s.env.Now(),
				},
			},
		},
		nil,
	)
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, "test").Once().Return(nil)

	s.env.OnActivity(activities.StorageSchedulesListActivity, mock.Anything, mock.Anything).Once().Return(
		&bunpaginate.Cursor[models.Schedule]{
			PageSize: 0,
			HasMore:  false,
			Data:     []models.Schedule{},
		},
		nil,
	)

	s.env.ExecuteWorkflow(RunTerminateSchedules, UninstallConnector{
		ConnectorID:       s.connectorID,
		DefaultWorkerName: "test",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_TerminateSchedules_StorageSchedulesList_Error() {
	s.env.OnActivity(activities.StorageSchedulesListActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunTerminateSchedules, UninstallConnector{
		ConnectorID:       s.connectorID,
		DefaultWorkerName: "test",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_TerminateSchedules_TemporalScheduleDelete_Error() {
	s.env.OnActivity(activities.StorageSchedulesListActivity, mock.Anything, mock.Anything).Once().Return(
		&bunpaginate.Cursor[models.Schedule]{
			PageSize: 1,
			HasMore:  false,
			Data: []models.Schedule{
				{
					ID:          "test",
					ConnectorID: s.connectorID,
					CreatedAt:   s.env.Now(),
				},
			},
		},
		nil,
	)
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, "test").Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunTerminateSchedules, UninstallConnector{
		ConnectorID:       s.connectorID,
		DefaultWorkerName: "test",
	})

	// Should log the error but continue
	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_TerminateSchedules__CursorError_Error() {
	s.env.OnActivity(activities.StorageSchedulesListActivity, mock.Anything, mock.Anything).Once().Return(
		&bunpaginate.Cursor[models.Schedule]{
			PageSize: 1,
			HasMore:  true,
			Next:     "toto",
			Data: []models.Schedule{
				{
					ID:          "test",
					ConnectorID: s.connectorID,
					CreatedAt:   s.env.Now(),
				},
			},
		},
		nil,
	)
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, "test").Once().Return(nil)

	s.env.ExecuteWorkflow(RunTerminateSchedules, UninstallConnector{
		ConnectorID:       s.connectorID,
		DefaultWorkerName: "test",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}
