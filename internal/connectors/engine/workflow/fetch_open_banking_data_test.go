package workflow

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_FetchOpenBankingData_Success() {
	psuID := uuid.New()
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	config := models.DefaultConfig()

	// Mock child workflows
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunFetchNextPayments, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	// Mock activity for updating last updated timestamp
	s.env.OnActivity(activities.StorageOpenBankingConnectionsLastUpdatedAtUpdateActivity, mock.Anything, psuID, connectorID, connectionID, mock.Anything).Once().Return(nil)

	// Mock send events workflow
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		FromPayload:  nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchOpenBankingData_WithFromPayload_Success() {
	psuID := uuid.New()
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	config := models.DefaultConfig()
	fromPayload := &FromPayload{
		ID:      "test-payload-id",
		Payload: []byte(`{"test": "data"}`),
	}

	// Mock child workflows
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunFetchNextPayments, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	// Mock activity for updating last updated timestamp
	s.env.OnActivity(activities.StorageOpenBankingConnectionsLastUpdatedAtUpdateActivity, mock.Anything, psuID, connectorID, connectionID, mock.Anything).Once().Return(nil)

	// Mock send events workflow
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		FromPayload:  fromPayload,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchOpenBankingData_RunFetchNextAccounts_Error() {
	psuID := uuid.New()
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	config := models.DefaultConfig()

	// Mock child workflow with error
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)
	s.env.OnWorkflow(RunFetchNextPayments, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	// Mock activity for updating last updated timestamp
	s.env.OnActivity(activities.StorageOpenBankingConnectionsLastUpdatedAtUpdateActivity, mock.Anything, psuID, connectorID, connectionID, mock.Anything).Once().Return(nil)

	// Mock send events workflow
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		FromPayload:  nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err) // Errors in child workflows are logged but don't fail the parent workflow
}

func (s *UnitTestSuite) Test_FetchOpenBankingData_RunFetchNextPayments_Error() {
	psuID := uuid.New()
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	config := models.DefaultConfig()

	// Mock child workflows
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunFetchNextPayments, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock activity for updating last updated timestamp
	s.env.OnActivity(activities.StorageOpenBankingConnectionsLastUpdatedAtUpdateActivity, mock.Anything, psuID, connectorID, connectionID, mock.Anything).Once().Return(nil)

	// Mock send events workflow
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		FromPayload:  nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err) // Errors in child workflows are logged but don't fail the parent workflow
}

func (s *UnitTestSuite) Test_FetchOpenBankingData_StoragePSUOpenBankingConnectionsLastUpdatedAtUpdate_Error() {
	psuID := uuid.New()
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	config := models.DefaultConfig()

	// Mock child workflows
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunFetchNextPayments, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	// Mock activity for updating last updated timestamp with error
	s.env.OnActivity(activities.StorageOpenBankingConnectionsLastUpdatedAtUpdateActivity, mock.Anything, psuID, connectorID, connectionID, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		FromPayload:  nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "updating open banking connection last updated at")
}

func (s *UnitTestSuite) Test_FetchOpenBankingData_RunSendEvents_Error() {
	psuID := uuid.New()
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	config := models.DefaultConfig()

	// Mock child workflows
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunFetchNextPayments, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	// Mock activity for updating last updated timestamp
	s.env.OnActivity(activities.StorageOpenBankingConnectionsLastUpdatedAtUpdateActivity, mock.Anything, psuID, connectorID, connectionID, mock.Anything).Once().Return(nil)

	// Mock send events workflow with error
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		FromPayload:  nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "sending events")
}

func (s *UnitTestSuite) Test_FetchOpenBankingData_BothChildWorkflows_Error() {
	psuID := uuid.New()
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	config := models.DefaultConfig()

	// Mock child workflows with errors
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test-accounts", "error-test", errors.New("error-test")),
	)
	s.env.OnWorkflow(RunFetchNextPayments, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test-payments", "error-test", errors.New("error-test")),
	)

	// Mock activity for updating last updated timestamp
	s.env.OnActivity(activities.StorageOpenBankingConnectionsLastUpdatedAtUpdateActivity, mock.Anything, psuID, connectorID, connectionID, mock.Anything).Once().Return(nil)

	// Mock send events workflow
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		FromPayload:  nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err) // Errors in child workflows are logged but don't fail the parent workflow
}
