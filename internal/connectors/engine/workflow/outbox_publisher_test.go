package workflow

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_RunOutboxPublisher_Success() {
	s.env.OnActivity(activities.OutboxPublishPendingEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunOutboxPublisher)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_RunOutboxPublisher_ActivityError() {
	expectedErr := temporal.NewNonRetryableApplicationError("error-test", "ACTIVITY", errors.New("error-test"))
	s.env.OnActivity(activities.OutboxPublishPendingEventsActivity, mock.Anything, mock.Anything).Once().Return(expectedErr)

	s.env.ExecuteWorkflow(RunOutboxPublisher)

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
	workflowErr, ok := err.(*temporal.WorkflowExecutionError)
	s.True(ok)
	s.ErrorContains(workflowErr.Unwrap(), expectedErr.Error())
}
