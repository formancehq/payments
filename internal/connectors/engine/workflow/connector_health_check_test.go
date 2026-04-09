package workflow

import (
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

// connectorWithUpdatedAt returns a *models.Connector for s.connectorID.
// Pass nil for updatedAt to simulate a connector that has never been updated.
func (s *UnitTestSuite) connectorWithUpdatedAt(updatedAt *time.Time) *models.Connector {
	return &models.Connector{
		ConnectorBase: models.ConnectorBase{ID: s.connectorID},
		UpdatedAt:     updatedAt,
	}
}

func (s *UnitTestSuite) Test_ConnectorHealthCheck_NoErrors_Success() {
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).
		Once().Return(s.connectorWithUpdatedAt(nil), nil)
	s.env.OnActivity(activities.StorageInstancesListSchedulesAboveErrorThresholdActivity, mock.Anything, s.connectorID, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Instance]{HasMore: false, Data: []models.Instance{}}, nil)

	s.env.ExecuteWorkflow(RunConnectorHealthCheck, ConnectorHealthCheck{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_ConnectorHealthCheck_PausesFetchSchedules_Success() {
	scheduleID := fmt.Sprintf("test-%s-FETCH_ACCOUNTS", s.connectorID.String())
	instances := []models.Instance{
		{ID: "wf-1", ScheduleID: scheduleID, ConnectorID: s.connectorID, Error: pointer.For("fetch error")},
	}

	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).
		Once().Return(s.connectorWithUpdatedAt(nil), nil)
	s.env.OnActivity(activities.StorageInstancesListSchedulesAboveErrorThresholdActivity, mock.Anything, s.connectorID, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Instance]{HasMore: false, Data: instances}, nil)
	s.env.OnActivity(activities.TemporalSchedulesPauseActivity, mock.Anything, mock.Anything).
		Once().Return(nil)

	s.env.ExecuteWorkflow(RunConnectorHealthCheck, ConnectorHealthCheck{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_ConnectorHealthCheck_AllCapabilities_Success() {
	instances := []models.Instance{
		{ID: "wf-1", ScheduleID: fmt.Sprintf("test-%s-FETCH_ACCOUNTS", s.connectorID.String()), ConnectorID: s.connectorID, Error: pointer.For("err")},
		{ID: "wf-2", ScheduleID: fmt.Sprintf("test-%s-FETCH_PAYMENTS", s.connectorID.String()), ConnectorID: s.connectorID, Error: pointer.For("err")},
		{ID: "wf-3", ScheduleID: fmt.Sprintf("test-%s-FETCH_EXTERNAL_ACCOUNTS", s.connectorID.String()), ConnectorID: s.connectorID, Error: pointer.For("err")},
		{ID: "wf-4", ScheduleID: fmt.Sprintf("test-%s-FETCH_BALANCES", s.connectorID.String()), ConnectorID: s.connectorID, Error: pointer.For("err")},
	}

	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).
		Once().Return(s.connectorWithUpdatedAt(nil), nil)
	s.env.OnActivity(activities.StorageInstancesListSchedulesAboveErrorThresholdActivity, mock.Anything, s.connectorID, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Instance]{HasMore: false, Data: instances}, nil)
	s.env.OnActivity(activities.TemporalSchedulesPauseActivity, mock.Anything, mock.Anything).
		Once().Return(nil)

	s.env.ExecuteWorkflow(RunConnectorHealthCheck, ConnectorHealthCheck{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_ConnectorHealthCheck_NonFetchSchedulesFiltered_Success() {
	// Instances whose schedule IDs do not contain any FETCH_ capability should be omitted.
	instances := []models.Instance{
		{ID: "wf-1", ScheduleID: fmt.Sprintf("test-%s-CREATE_PAYOUT", s.connectorID.String()), ConnectorID: s.connectorID, Error: pointer.For("err")},
		{ID: "wf-2", ScheduleID: fmt.Sprintf("test-%s-CREATE_TRANSFER", s.connectorID.String()), ConnectorID: s.connectorID, Error: pointer.For("err")},
	}

	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).
		Once().Return(s.connectorWithUpdatedAt(nil), nil)
	s.env.OnActivity(activities.StorageInstancesListSchedulesAboveErrorThresholdActivity, mock.Anything, s.connectorID, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Instance]{HasMore: false, Data: instances}, nil)
	// TemporalSchedulesPauseActivity must NOT be called.

	s.env.ExecuteWorkflow(RunConnectorHealthCheck, ConnectorHealthCheck{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_ConnectorHealthCheck_PartialFilter_Success() {
	// Mix: only FETCH_ instances should reach TemporalSchedulesPause.
	instances := []models.Instance{
		{ID: "wf-1", ScheduleID: fmt.Sprintf("test-%s-FETCH_PAYMENTS", s.connectorID.String()), ConnectorID: s.connectorID, Error: pointer.For("err")},
		{ID: "wf-2", ScheduleID: fmt.Sprintf("test-%s-CREATE_PAYOUT", s.connectorID.String()), ConnectorID: s.connectorID, Error: pointer.For("err")},
	}

	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).
		Once().Return(s.connectorWithUpdatedAt(nil), nil)
	s.env.OnActivity(activities.StorageInstancesListSchedulesAboveErrorThresholdActivity, mock.Anything, s.connectorID, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Instance]{HasMore: false, Data: instances}, nil)
	s.env.OnActivity(activities.TemporalSchedulesPauseActivity, mock.Anything, mock.Anything).
		Once().Return(nil)

	s.env.ExecuteWorkflow(RunConnectorHealthCheck, ConnectorHealthCheck{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_ConnectorHealthCheck_HasMore_Success() {
	scheduleID := fmt.Sprintf("test-%s-FETCH_ACCOUNTS", s.connectorID.String())

	nextCursor := bunpaginate.EncodeCursor(
		bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[storage.InstanceQuery]]{
			Offset:   1,
			Order:    bunpaginate.OrderAsc,
			PageSize: 1,
		},
	)

	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).
		Once().Return(s.connectorWithUpdatedAt(nil), nil)
	s.env.OnActivity(activities.StorageInstancesListSchedulesAboveErrorThresholdActivity, mock.Anything, s.connectorID, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Instance]{
			HasMore: true,
			Next:    nextCursor,
			Data: []models.Instance{
				{ID: "wf-1", ScheduleID: scheduleID, ConnectorID: s.connectorID, Error: pointer.For("err")},
			},
		}, nil)
	s.env.OnActivity(activities.TemporalSchedulesPauseActivity, mock.Anything, mock.Anything).
		Once().Return(nil)

	s.env.OnActivity(activities.StorageInstancesListSchedulesAboveErrorThresholdActivity, mock.Anything, s.connectorID, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Instance]{HasMore: false, Data: []models.Instance{}}, nil)

	s.env.ExecuteWorkflow(RunConnectorHealthCheck, ConnectorHealthCheck{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_ConnectorHealthCheck_StorageConnectorsGet_Error() {
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).
		Once().Return(nil, temporal.NewNonRetryableApplicationError("connector not found", "STORAGE", errors.New("connector not found")))

	s.env.ExecuteWorkflow(RunConnectorHealthCheck, ConnectorHealthCheck{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "connector not found")
}

func (s *UnitTestSuite) Test_ConnectorHealthCheck_StorageInstancesListSchedulesAboveErrorThreshold_Error() {
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).
		Once().Return(s.connectorWithUpdatedAt(nil), nil)
	s.env.OnActivity(activities.StorageInstancesListSchedulesAboveErrorThresholdActivity, mock.Anything, s.connectorID, mock.Anything).
		Once().Return(
			nil,
			temporal.NewNonRetryableApplicationError("storage error", "storage error", errors.New("storage error")),
		)

	s.env.ExecuteWorkflow(RunConnectorHealthCheck, ConnectorHealthCheck{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "storage error")
}

func (s *UnitTestSuite) Test_ConnectorHealthCheck_TemporalSchedulesPause_Error() {
	scheduleID := fmt.Sprintf("test-%s-FETCH_BALANCES", s.connectorID.String())

	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).
		Once().Return(s.connectorWithUpdatedAt(nil), nil)
	s.env.OnActivity(activities.StorageInstancesListSchedulesAboveErrorThresholdActivity, mock.Anything, s.connectorID, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Instance]{
			HasMore: false,
			Data: []models.Instance{
				{ID: "wf-1", ScheduleID: scheduleID, ConnectorID: s.connectorID, Error: pointer.For("err")},
			},
		}, nil)
	s.env.OnActivity(activities.TemporalSchedulesPauseActivity, mock.Anything, mock.Anything).
		Once().Return(
			temporal.NewNonRetryableApplicationError("pause error", "pause error", errors.New("pause error")),
		)

	s.env.ExecuteWorkflow(RunConnectorHealthCheck, ConnectorHealthCheck{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "pause error")
}

// --- UpdatedAt filtering tests ---

func (s *UnitTestSuite) Test_ConnectorHealthCheck_UpdatedAt_NilConnectorUpdatedAt_InstanceIncluded() {
	// connector.UpdatedAt == nil → no filtering, all FETCH_ instances are included
	scheduleID := fmt.Sprintf("test-%s-FETCH_ACCOUNTS", s.connectorID.String())
	now := s.env.Now()
	instance := models.Instance{
		ID: "wf-1", ScheduleID: scheduleID, ConnectorID: s.connectorID,
		CreatedAt: now.Add(-time.Hour),
		Error:     pointer.For("err"),
	}

	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).
		Once().Return(s.connectorWithUpdatedAt(nil), nil)
	s.env.OnActivity(activities.StorageInstancesListSchedulesAboveErrorThresholdActivity, mock.Anything, s.connectorID, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Instance]{HasMore: false, Data: []models.Instance{instance}}, nil)
	s.env.OnActivity(activities.TemporalSchedulesPauseActivity, mock.Anything, mock.Anything).
		Once().Return(nil)

	s.env.ExecuteWorkflow(RunConnectorHealthCheck, ConnectorHealthCheck{ConnectorID: s.connectorID})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_ConnectorHealthCheck_UpdatedAt_InstanceAfterUpdate_IncludedInPause() {
	// instance.CreatedAt > connector.UpdatedAt → include in toPause
	scheduleID := fmt.Sprintf("test-%s-FETCH_PAYMENTS", s.connectorID.String())
	now := s.env.Now()
	updatedAt := now.Add(-time.Hour)
	instance := models.Instance{
		ID: "wf-1", ScheduleID: scheduleID, ConnectorID: s.connectorID,
		CreatedAt: now, // after the config update
		Error:     pointer.For("err"),
	}

	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).
		Once().Return(s.connectorWithUpdatedAt(&updatedAt), nil)
	s.env.OnActivity(activities.StorageInstancesListSchedulesAboveErrorThresholdActivity, mock.Anything, s.connectorID, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Instance]{HasMore: false, Data: []models.Instance{instance}}, nil)
	s.env.OnActivity(activities.TemporalSchedulesPauseActivity, mock.Anything, mock.Anything).
		Once().Return(nil)

	s.env.ExecuteWorkflow(RunConnectorHealthCheck, ConnectorHealthCheck{ConnectorID: s.connectorID})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_ConnectorHealthCheck_UpdatedAt_InstanceBeforeUpdate_ExcludedFromPause() {
	// instance.CreatedAt < connector.UpdatedAt → skip; TemporalSchedulesPause must NOT be called
	scheduleID := fmt.Sprintf("test-%s-FETCH_ACCOUNTS", s.connectorID.String())
	now := s.env.Now()
	updatedAt := now
	instance := models.Instance{
		ID: "wf-1", ScheduleID: scheduleID, ConnectorID: s.connectorID,
		CreatedAt: now.Add(-time.Hour), // before the config update
		Error:     pointer.For("err"),
	}

	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).
		Once().Return(s.connectorWithUpdatedAt(&updatedAt), nil)
	s.env.OnActivity(activities.StorageInstancesListSchedulesAboveErrorThresholdActivity, mock.Anything, s.connectorID, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Instance]{HasMore: false, Data: []models.Instance{instance}}, nil)
	// TemporalSchedulesPauseActivity must NOT be called.

	s.env.ExecuteWorkflow(RunConnectorHealthCheck, ConnectorHealthCheck{ConnectorID: s.connectorID})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_ConnectorHealthCheck_UpdatedAt_InstanceEqualToUpdate_ExcludedFromPause() {
	// instance.CreatedAt == connector.UpdatedAt → skip; TemporalSchedulesPause must NOT be called
	scheduleID := fmt.Sprintf("test-%s-FETCH_ACCOUNTS", s.connectorID.String())
	now := s.env.Now()
	instance := models.Instance{
		ID: "wf-1", ScheduleID: scheduleID, ConnectorID: s.connectorID,
		CreatedAt: now, // equal to the config update timestamp
		Error:     pointer.For("err"),
	}

	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).
		Once().Return(s.connectorWithUpdatedAt(&now), nil)
	s.env.OnActivity(activities.StorageInstancesListSchedulesAboveErrorThresholdActivity, mock.Anything, s.connectorID, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Instance]{HasMore: false, Data: []models.Instance{instance}}, nil)
	// TemporalSchedulesPauseActivity must NOT be called.

	s.env.ExecuteWorkflow(RunConnectorHealthCheck, ConnectorHealthCheck{ConnectorID: s.connectorID})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_ConnectorHealthCheck_UpdatedAt_MixedInstances_OnlyNewerOnes() {
	// One instance before the config update (excluded), one after (included).
	scheduleID := fmt.Sprintf("test-%s-FETCH_BALANCES", s.connectorID.String())
	now := s.env.Now()
	updatedAt := now.Add(-30 * time.Minute)

	old := models.Instance{
		ID: "wf-old", ScheduleID: scheduleID, ConnectorID: s.connectorID,
		CreatedAt: now.Add(-time.Hour), // before config update
		Error:     pointer.For("old error"),
	}
	recent := models.Instance{
		ID: "wf-recent", ScheduleID: scheduleID, ConnectorID: s.connectorID,
		CreatedAt: now, // after config update
		Error:     pointer.For("new error"),
	}

	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).
		Once().Return(s.connectorWithUpdatedAt(&updatedAt), nil)
	s.env.OnActivity(activities.StorageInstancesListSchedulesAboveErrorThresholdActivity, mock.Anything, s.connectorID, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Instance]{HasMore: false, Data: []models.Instance{old, recent}}, nil)
	s.env.OnActivity(activities.TemporalSchedulesPauseActivity, mock.Anything, mock.Anything).
		Once().Return(nil)

	s.env.ExecuteWorkflow(RunConnectorHealthCheck, ConnectorHealthCheck{ConnectorID: s.connectorID})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}
