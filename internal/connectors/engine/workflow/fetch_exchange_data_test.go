package workflow

import (
	"errors"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/workflow"
)

func (s *UnitTestSuite) Test_FetchExchangeData_AllSuccess() {
	// Mock all three child workflows completing successfully
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchNextAccounts, nextTasks []models.ConnectorTaskTree) error {
		s.Equal(s.connectorID, req.ConnectorID)
		s.False(req.Periodically)
		// Verify next tasks include FETCH_BALANCES
		s.Equal(1, len(nextTasks))
		s.Equal(models.TASK_FETCH_BALANCES, nextTasks[0].TaskType)
		return nil
	})
	s.env.OnWorkflow(RunFetchNextOrders, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchNextOrders, nextTasks []models.ConnectorTaskTree) error {
		s.Equal(s.connectorID, req.ConnectorID)
		s.False(req.Periodically)
		return nil
	})
	s.env.OnWorkflow(RunFetchNextConversions, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchNextConversions, nextTasks []models.ConnectorTaskTree) error {
		s.Equal(s.connectorID, req.ConnectorID)
		s.False(req.Periodically)
		return nil
	})

	s.env.ExecuteWorkflow(RunFetchExchangeData, FetchExchangeData{
		ConnectorID: s.connectorID,
		FromPayload: &FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchExchangeData_AccountsFetchError() {
	// Mock accounts fetch failing, others succeed
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchNextAccounts, nextTasks []models.ConnectorTaskTree) error {
		return errors.New("accounts fetch failed")
	})
	s.env.OnWorkflow(RunFetchNextOrders, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchNextOrders, nextTasks []models.ConnectorTaskTree) error {
		return nil
	})
	s.env.OnWorkflow(RunFetchNextConversions, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchNextConversions, nextTasks []models.ConnectorTaskTree) error {
		return nil
	})

	s.env.ExecuteWorkflow(RunFetchExchangeData, FetchExchangeData{
		ConnectorID: s.connectorID,
		FromPayload: &FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "accounts fetch failed")
}

func (s *UnitTestSuite) Test_FetchExchangeData_OrdersFetchError() {
	// Mock orders fetch failing, others succeed
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchNextAccounts, nextTasks []models.ConnectorTaskTree) error {
		return nil
	})
	s.env.OnWorkflow(RunFetchNextOrders, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchNextOrders, nextTasks []models.ConnectorTaskTree) error {
		return errors.New("orders fetch failed")
	})
	s.env.OnWorkflow(RunFetchNextConversions, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchNextConversions, nextTasks []models.ConnectorTaskTree) error {
		return nil
	})

	s.env.ExecuteWorkflow(RunFetchExchangeData, FetchExchangeData{
		ConnectorID: s.connectorID,
		FromPayload: &FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "orders fetch failed")
}

func (s *UnitTestSuite) Test_FetchExchangeData_ConversionsFetchError() {
	// Mock conversions fetch failing, others succeed
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchNextAccounts, nextTasks []models.ConnectorTaskTree) error {
		return nil
	})
	s.env.OnWorkflow(RunFetchNextOrders, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchNextOrders, nextTasks []models.ConnectorTaskTree) error {
		return nil
	})
	s.env.OnWorkflow(RunFetchNextConversions, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchNextConversions, nextTasks []models.ConnectorTaskTree) error {
		return errors.New("conversions fetch failed")
	})

	s.env.ExecuteWorkflow(RunFetchExchangeData, FetchExchangeData{
		ConnectorID: s.connectorID,
		FromPayload: &FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "conversions fetch failed")
}

func (s *UnitTestSuite) Test_FetchExchangeData_WithPayload_Success() {
	// Verify payload is passed correctly to child workflows
	testPayload := []byte(`{"accountId":"test-123"}`)

	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchNextAccounts, nextTasks []models.ConnectorTaskTree) error {
		s.Equal(string(testPayload), string(req.FromPayload.Payload))
		s.Equal("test-id", req.FromPayload.ID)
		return nil
	})
	s.env.OnWorkflow(RunFetchNextOrders, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchNextOrders, nextTasks []models.ConnectorTaskTree) error {
		s.Equal(string(testPayload), string(req.FromPayload.Payload))
		s.Equal("test-id", req.FromPayload.ID)
		return nil
	})
	s.env.OnWorkflow(RunFetchNextConversions, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchNextConversions, nextTasks []models.ConnectorTaskTree) error {
		s.Equal(string(testPayload), string(req.FromPayload.Payload))
		s.Equal("test-id", req.FromPayload.ID)
		return nil
	})

	s.env.ExecuteWorkflow(RunFetchExchangeData, FetchExchangeData{
		ConnectorID: s.connectorID,
		FromPayload: &FromPayload{
			ID:      "test-id",
			Payload: testPayload,
		},
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}
