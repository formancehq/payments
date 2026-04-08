package workflow

import (
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
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
	scheduleID := fmt.Sprintf("test-%s-FETCH_ACCOUNTS", s.connectorID.String())
	schedule := models.Schedule{ID: scheduleID, ConnectorID: s.connectorID}

	s.env.OnActivity(activities.StorageSchedulesListActivity, mock.Anything, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Schedule]{HasMore: false, Data: []models.Schedule{schedule}}, nil)
	s.env.OnActivity(activities.TemporalScheduleUpdatePollingPeriodActivity, mock.Anything, scheduleID, mock.Anything).
		Once().Return(nil)

	s.env.ExecuteWorkflow(RunUpdateSchedulePollingPeriod, UpdateSchedulePollingPeriod{
		ConnectorID: s.connectorID,
		Config:      models.Config{PollingPeriod: time.Hour},
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_UpdateSchedulePollingPeriod_NonFetchSchedule_Skipped() {
	// Schedules without a FETCH_ capability keyword must be ignored entirely.
	nonFetch := models.Schedule{ID: fmt.Sprintf("test-%s-HEALTH_CHECK", s.connectorID.String()), ConnectorID: s.connectorID}

	s.env.OnActivity(activities.StorageSchedulesListActivity, mock.Anything, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Schedule]{HasMore: false, Data: []models.Schedule{nonFetch}}, nil)
	// Neither TemporalSchedulesUnpause nor TemporalScheduleUpdatePollingPeriod should be called.

	s.env.ExecuteWorkflow(RunUpdateSchedulePollingPeriod, UpdateSchedulePollingPeriod{
		ConnectorID: s.connectorID,
		Config:      models.Config{PollingPeriod: time.Hour},
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_UpdateSchedulePollingPeriod_PausedSchedule_UnpausedAndUpdated() {
	scheduleID := fmt.Sprintf("test-%s-FETCH_PAYMENTS", s.connectorID.String())
	pausedAt := s.env.Now().UTC()
	schedule := models.Schedule{
		ID:          scheduleID,
		ConnectorID: s.connectorID,
		PausedAt:    pointer.For(pausedAt),
	}

	s.env.OnActivity(activities.StorageSchedulesListActivity, mock.Anything, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Schedule]{HasMore: false, Data: []models.Schedule{schedule}}, nil)
	s.env.OnActivity(activities.TemporalSchedulesUnpauseActivity, mock.Anything, []models.Schedule{schedule}).
		Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleUpdatePollingPeriodActivity, mock.Anything, scheduleID, mock.Anything).
		Once().Return(nil)

	s.env.ExecuteWorkflow(RunUpdateSchedulePollingPeriod, UpdateSchedulePollingPeriod{
		ConnectorID: s.connectorID,
		Config:      models.Config{PollingPeriod: time.Hour},
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_UpdateSchedulePollingPeriod_MixedPaused_OnlyPausedUnpaused() {
	pausedAt := s.env.Now().UTC()
	pausedID := fmt.Sprintf("test-%s-FETCH_ACCOUNTS", s.connectorID.String())
	activeID := fmt.Sprintf("test-%s-FETCH_PAYMENTS", s.connectorID.String())
	paused := models.Schedule{ID: pausedID, ConnectorID: s.connectorID, PausedAt: pointer.For(pausedAt)}
	active := models.Schedule{ID: activeID, ConnectorID: s.connectorID}

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
	scheduleID := fmt.Sprintf("test-%s-FETCH_BALANCES", s.connectorID.String())
	pausedAt := s.env.Now()
	schedule := models.Schedule{
		ID:          scheduleID,
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
