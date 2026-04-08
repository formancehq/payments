package workflow

import (
	"errors"
	"fmt"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_ConnectorHealthCheck_NoErrors_Success() {
	s.env.OnActivity(activities.StorageInstancesGetScheduleErrorsActivity, mock.Anything, s.connectorID, mock.Anything).
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

	s.env.OnActivity(activities.StorageInstancesGetScheduleErrorsActivity, mock.Anything, s.connectorID, mock.Anything).
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

	s.env.OnActivity(activities.StorageInstancesGetScheduleErrorsActivity, mock.Anything, s.connectorID, mock.Anything).
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

	s.env.OnActivity(activities.StorageInstancesGetScheduleErrorsActivity, mock.Anything, s.connectorID, mock.Anything).
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

	s.env.OnActivity(activities.StorageInstancesGetScheduleErrorsActivity, mock.Anything, s.connectorID, mock.Anything).
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

	s.env.OnActivity(activities.StorageInstancesGetScheduleErrorsActivity, mock.Anything, s.connectorID, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Instance]{
			HasMore: true,
			Next:    nextCursor,
			Data: []models.Instance{
				{ID: "wf-1", ScheduleID: scheduleID, ConnectorID: s.connectorID, Error: pointer.For("err")},
			},
		}, nil)
	s.env.OnActivity(activities.TemporalSchedulesPauseActivity, mock.Anything, mock.Anything).
		Once().Return(nil)

	s.env.OnActivity(activities.StorageInstancesGetScheduleErrorsActivity, mock.Anything, s.connectorID, mock.Anything).
		Once().Return(&bunpaginate.Cursor[models.Instance]{HasMore: false, Data: []models.Instance{}}, nil)

	s.env.ExecuteWorkflow(RunConnectorHealthCheck, ConnectorHealthCheck{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_ConnectorHealthCheck_StorageInstancesGetScheduleErrors_Error() {
	s.env.OnActivity(activities.StorageInstancesGetScheduleErrorsActivity, mock.Anything, s.connectorID, mock.Anything).
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

	s.env.OnActivity(activities.StorageInstancesGetScheduleErrorsActivity, mock.Anything, s.connectorID, mock.Anything).
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
