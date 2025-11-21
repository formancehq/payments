package workflow

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_RunOutboxCleanup_Success() {
	s.env.OnActivity(activities.OutboxDeleteOldProcessedEventsActivity, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunOutboxCleanup)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_RunOutboxCleanup_ActivityError() {
	expectedErr := temporal.NewNonRetryableApplicationError("error-test", "ACTIVITY", errors.New("error-test"))
	s.env.OnActivity(activities.OutboxDeleteOldProcessedEventsActivity, mock.Anything).Once().Return(expectedErr)

	s.env.ExecuteWorkflow(RunOutboxCleanup)

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
	var workflowErr *temporal.WorkflowExecutionError
	ok := errors.As(err, &workflowErr)
	s.True(ok)
	s.ErrorContains(workflowErr.Unwrap(), expectedErr.Error())
}
