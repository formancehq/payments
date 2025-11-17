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
	dataToFetch := []models.OpenBankingDataToFetch{
		models.OpenBankingDataToFetchAccountsAndBalances,
		models.OpenBankingDataToFetchPayments,
	}

	// Mock child workflows
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunFetchNextPayments, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	// Mock activity for updating last updated timestamp
	s.env.OnActivity(activities.StorageOpenBankingConnectionsLastUpdatedAtUpdateActivity, mock.Anything, psuID, connectorID, connectionID, mock.Anything).Once().Return(nil)

	// Mock send events workflow

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		DataToFetch:  dataToFetch,
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
	dataToFetch := []models.OpenBankingDataToFetch{
		models.OpenBankingDataToFetchAccountsAndBalances,
		models.OpenBankingDataToFetchPayments,
	}

	// Mock child workflows
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunFetchNextPayments, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	// Mock activity for updating last updated timestamp
	s.env.OnActivity(activities.StorageOpenBankingConnectionsLastUpdatedAtUpdateActivity, mock.Anything, psuID, connectorID, connectionID, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		DataToFetch:  dataToFetch,
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
	dataToFetch := []models.OpenBankingDataToFetch{
		models.OpenBankingDataToFetchAccountsAndBalances,
		models.OpenBankingDataToFetchPayments,
	}

	// Mock child workflow with error
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)
	s.env.OnWorkflow(RunFetchNextPayments, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	// Mock activity for updating last updated timestamp should not be called
	s.env.OnActivity(activities.StorageOpenBankingConnectionsLastUpdatedAtUpdateActivity, mock.Anything, psuID, connectorID, connectionID, mock.Anything).Never().Return(nil)

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		DataToFetch:  dataToFetch,
		FromPayload:  nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err) // Child workflow errors now fail the parent workflow
	s.ErrorContains(err, "failed to fetch accounts")
}

func (s *UnitTestSuite) Test_FetchOpenBankingData_RunFetchNextPayments_Error() {
	psuID := uuid.New()
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	config := models.DefaultConfig()
	dataToFetch := []models.OpenBankingDataToFetch{
		models.OpenBankingDataToFetchAccountsAndBalances,
		models.OpenBankingDataToFetchPayments,
	}

	// Mock child workflows
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunFetchNextPayments, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock activity for updating last updated timestamp should not be called
	s.env.OnActivity(activities.StorageOpenBankingConnectionsLastUpdatedAtUpdateActivity, mock.Anything, psuID, connectorID, connectionID, mock.Anything).Never().Return(nil)

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		DataToFetch:  dataToFetch,
		FromPayload:  nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err) // Child workflow errors now fail the parent workflow
	s.ErrorContains(err, "failed to fetch payments")
}

func (s *UnitTestSuite) Test_FetchOpenBankingData_StoragePSUOpenBankingConnectionsLastUpdatedAtUpdate_Error() {
	psuID := uuid.New()
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	config := models.DefaultConfig()
	dataToFetch := []models.OpenBankingDataToFetch{
		models.OpenBankingDataToFetchAccountsAndBalances,
		models.OpenBankingDataToFetchPayments,
	}

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
		DataToFetch:  dataToFetch,
		FromPayload:  nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "updating open banking connection last updated at")
}

func (s *UnitTestSuite) Test_FetchOpenBankingData_BothChildWorkflows_Error() {
	psuID := uuid.New()
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	config := models.DefaultConfig()
	dataToFetch := []models.OpenBankingDataToFetch{
		models.OpenBankingDataToFetchAccountsAndBalances,
		models.OpenBankingDataToFetchPayments,
	}

	// Mock child workflows with errors
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test-accounts", "error-test", errors.New("error-test")),
	)
	s.env.OnWorkflow(RunFetchNextPayments, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test-payments", "error-test", errors.New("error-test")),
	)

	// Mock activity for updating last updated timestamp should not be called
	s.env.OnActivity(activities.StorageOpenBankingConnectionsLastUpdatedAtUpdateActivity, mock.Anything, psuID, connectorID, connectionID, mock.Anything).Never().Return(nil)

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		DataToFetch:  dataToFetch,
		FromPayload:  nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err) // Child workflow errors now fail the parent workflow
	// Should contain error message for accounts (first error checked)
	s.ErrorContains(err, "failed to fetch accounts")
}

func (s *UnitTestSuite) Test_FetchOpenBankingData_EmptyDataToFetch_Error() {
	psuID := uuid.New()
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	config := models.DefaultConfig()
	dataToFetch := []models.OpenBankingDataToFetch{} // Empty array

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		DataToFetch:  dataToFetch,
		FromPayload:  nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "no data to fetch")
}

func (s *UnitTestSuite) Test_FetchOpenBankingData_AccountsAndBalances_Success() {
	psuID := uuid.New()
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	config := models.DefaultConfig()
	dataToFetch := []models.OpenBankingDataToFetch{
		models.OpenBankingDataToFetchAccountsAndBalances,
	}

	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	// RunFetchNextBalances is only called via a subworkflow
	s.env.OnWorkflow(RunFetchNextBalances, mock.Anything, mock.Anything, mock.Anything).Never().Return(nil)

	// Mock activity for updating last updated timestamp
	s.env.OnActivity(activities.StorageOpenBankingConnectionsLastUpdatedAtUpdateActivity, mock.Anything, psuID, connectorID, connectionID, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		DataToFetch:  dataToFetch,
		FromPayload:  nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchOpenBankingData_AccountsAndBalances_Error() {
	psuID := uuid.New()
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	config := models.DefaultConfig()
	dataToFetch := []models.OpenBankingDataToFetch{
		models.OpenBankingDataToFetchAccountsAndBalances,
	}

	// Mock child workflow with error
	s.env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock activity for updating last updated timestamp should not be called
	s.env.OnActivity(activities.StorageOpenBankingConnectionsLastUpdatedAtUpdateActivity, mock.Anything, psuID, connectorID, connectionID, mock.Anything).Never().Return(nil)

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		DataToFetch:  dataToFetch,
		FromPayload:  nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err) // Child workflow errors now fail the parent workflow
	s.ErrorContains(err, "failed to fetch accounts")
}

func (s *UnitTestSuite) Test_FetchOpenBankingData_PaymentsOnly_Success() {
	psuID := uuid.New()
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	config := models.DefaultConfig()
	dataToFetch := []models.OpenBankingDataToFetch{
		models.OpenBankingDataToFetchPayments,
	}

	// Mock child workflow
	s.env.OnWorkflow(RunFetchNextPayments, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	// Mock activity for updating last updated timestamp
	s.env.OnActivity(activities.StorageOpenBankingConnectionsLastUpdatedAtUpdateActivity, mock.Anything, psuID, connectorID, connectionID, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		DataToFetch:  dataToFetch,
		FromPayload:  nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchOpenBankingData_PaymentsOnly_Error() {
	psuID := uuid.New()
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	config := models.DefaultConfig()
	dataToFetch := []models.OpenBankingDataToFetch{
		models.OpenBankingDataToFetchPayments,
	}

	// Mock child workflow with error
	s.env.OnWorkflow(RunFetchNextPayments, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock activity for updating last updated timestamp should not be called
	s.env.OnActivity(activities.StorageOpenBankingConnectionsLastUpdatedAtUpdateActivity, mock.Anything, psuID, connectorID, connectionID, mock.Anything).Never().Return(nil)

	s.env.ExecuteWorkflow(RunFetchOpenBankingData, FetchOpenBankingData{
		PsuID:        psuID,
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		Config:       config,
		DataToFetch:  dataToFetch,
		FromPayload:  nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err) // Child workflow errors now fail the parent workflow
	s.ErrorContains(err, "failed to fetch payments")
}
