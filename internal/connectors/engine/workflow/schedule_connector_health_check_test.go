package workflow

import (
	"errors"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_ScheduleConnectorHealthCheck_Success() {
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).
		Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.Anything).
		Once().Return(nil)

	s.env.ExecuteWorkflow(RunScheduleConnectorHealthCheck, ScheduleConnectorHealthCheck{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_ScheduleConnectorHealthCheck_ScheduleID_Format() {
	expectedScheduleID := fmt.Sprintf("test-%s-HEALTH_CHECK", s.connectorID.String())

	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.MatchedBy(func(schedule models.Schedule) bool {
		return schedule.ID == expectedScheduleID && schedule.ConnectorID == s.connectorID
	})).Once().Return(nil)

	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.MatchedBy(func(opts activities.ScheduleCreateOptions) bool {
		return opts.ScheduleID == expectedScheduleID
	})).Once().Return(nil)

	s.env.ExecuteWorkflow(RunScheduleConnectorHealthCheck, ScheduleConnectorHealthCheck{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_ScheduleConnectorHealthCheck_SearchAttributes() {
	expectedScheduleID := fmt.Sprintf("test-%s-HEALTH_CHECK", s.connectorID.String())

	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).
		Once().Return(nil)

	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.MatchedBy(func(opts activities.ScheduleCreateOptions) bool {
		scheduleIDAttr, hasScheduleID := opts.SearchAttributes[SearchAttributeScheduleID]
		stackAttr, hasStack := opts.SearchAttributes[SearchAttributeStack]
		return hasScheduleID && scheduleIDAttr == expectedScheduleID &&
			hasStack && stackAttr == "test"
	})).Once().Return(nil)

	s.env.ExecuteWorkflow(RunScheduleConnectorHealthCheck, ScheduleConnectorHealthCheck{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_ScheduleConnectorHealthCheck_StorageSchedulesStore_Error() {
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).
		Once().Return(temporal.NewNonRetryableApplicationError("storage error", "STORAGE", errors.New("storage error")))

	s.env.ExecuteWorkflow(RunScheduleConnectorHealthCheck, ScheduleConnectorHealthCheck{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "storage error")
}

func (s *UnitTestSuite) Test_ScheduleConnectorHealthCheck_TemporalScheduleCreate_Error() {
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).
		Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.Anything).
		Once().Return(temporal.NewNonRetryableApplicationError("temporal error", "TEMPORAL", errors.New("temporal error")))

	s.env.ExecuteWorkflow(RunScheduleConnectorHealthCheck, ScheduleConnectorHealthCheck{
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "temporal error")
}
