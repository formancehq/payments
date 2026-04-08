package workflow

import (
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_UpdateSchedulePollingPeriod_NoSchedules_Success() {
	s.env.OnActivity(activities.StorageSchedulesListActivity, mock.Anything, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Schedule]{HasMore: false, Data: []models.Schedule{}}, nil)

	s.env.ExecuteWorkflow(RunUpdateSchedulePollingPeriod, UpdateSchedulePollingPeriod{
		ConnectorID: s.connectorID,
		Config:      models.Config{PollingPeriod: time.Hour},
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_UpdateSchedulePollingPeriod_NotPaused_Success() {
	schedule := models.Schedule{ID: "sched-1", ConnectorID: s.connectorID}

	s.env.OnActivity(activities.StorageSchedulesListActivity, mock.Anything, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Schedule]{HasMore: false, Data: []models.Schedule{schedule}}, nil)
	s.env.OnActivity(activities.TemporalScheduleUpdatePollingPeriodActivity, mock.Anything, schedule.ID, mock.Anything).
		Once().Return(nil)

	s.env.ExecuteWorkflow(RunUpdateSchedulePollingPeriod, UpdateSchedulePollingPeriod{
		ConnectorID: s.connectorID,
		Config:      models.Config{PollingPeriod: time.Hour},
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_UpdateSchedulePollingPeriod_PausedSchedule_UnpausedAndUpdated() {
	pausedAt := s.env.Now()
	schedule := models.Schedule{
		ID:          "sched-paused",
		ConnectorID: s.connectorID,
		PausedAt:    pointer.For(pausedAt),
	}

	s.env.OnActivity(activities.StorageSchedulesListActivity, mock.Anything, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Schedule]{HasMore: false, Data: []models.Schedule{schedule}}, nil)
	s.env.OnActivity(activities.TemporalSchedulesUnpauseActivity, mock.Anything, []models.Schedule{schedule}).
		Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleUpdatePollingPeriodActivity, mock.Anything, schedule.ID, mock.Anything).
		Once().Return(nil)

	s.env.ExecuteWorkflow(RunUpdateSchedulePollingPeriod, UpdateSchedulePollingPeriod{
		ConnectorID: s.connectorID,
		Config:      models.Config{PollingPeriod: time.Hour},
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_UpdateSchedulePollingPeriod_MixedPaused_OnlyPausedUnpaused() {
	pausedAt := s.env.Now()
	paused := models.Schedule{ID: "sched-paused", ConnectorID: s.connectorID, PausedAt: pointer.For(pausedAt)}
	active := models.Schedule{ID: "sched-active", ConnectorID: s.connectorID}

	s.env.OnActivity(activities.StorageSchedulesListActivity, mock.Anything, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Schedule]{HasMore: false, Data: []models.Schedule{paused, active}}, nil)
	s.env.OnActivity(activities.TemporalSchedulesUnpauseActivity, mock.Anything, []models.Schedule{paused}).
		Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleUpdatePollingPeriodActivity, mock.Anything, mock.Anything, mock.Anything).
		Times(2).Return(nil)

	s.env.ExecuteWorkflow(RunUpdateSchedulePollingPeriod, UpdateSchedulePollingPeriod{
		ConnectorID: s.connectorID,
		Config:      models.Config{PollingPeriod: time.Hour},
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_UpdateSchedulePollingPeriod_TemporalUnpause_Error() {
	pausedAt := s.env.Now()
	schedule := models.Schedule{
		ID:          "sched-paused",
		ConnectorID: s.connectorID,
		PausedAt:    pointer.For(pausedAt),
	}

	s.env.OnActivity(activities.StorageSchedulesListActivity, mock.Anything, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Schedule]{HasMore: false, Data: []models.Schedule{schedule}}, nil)
	s.env.OnActivity(activities.TemporalSchedulesUnpauseActivity, mock.Anything, mock.Anything).
		Once().Return(temporal.NewNonRetryableApplicationError("unpause error", "TEMPORAL", errors.New("unpause error")))

	s.env.ExecuteWorkflow(RunUpdateSchedulePollingPeriod, UpdateSchedulePollingPeriod{
		ConnectorID: s.connectorID,
		Config:      models.Config{PollingPeriod: time.Hour},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "unpause error")
}

func (s *UnitTestSuite) Test_UpdateSchedulePollingPeriod_StorageSchedulesList_Error() {
	s.env.OnActivity(activities.StorageSchedulesListActivity, mock.Anything, mock.Anything).
		Once().Return(nil, temporal.NewNonRetryableApplicationError("storage error", "STORAGE", errors.New("storage error")))

	s.env.ExecuteWorkflow(RunUpdateSchedulePollingPeriod, UpdateSchedulePollingPeriod{
		ConnectorID: s.connectorID,
		Config:      models.Config{PollingPeriod: time.Hour},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "storage error")
}
