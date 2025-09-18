package workflow

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromConnectionID_Success() {
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsDeleteFromConnectionIDActivity, mock.Anything, psuID, connectorID, connectionID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromConnectionIDActivity, mock.Anything, psuID, connectorID, connectionID).Once().Return(nil)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		FromConnectionID: &DeleteOpenBankingConnectionDataFromConnectionID{
			PSUID:        psuID,
			ConnectorID:  connectorID,
			ConnectionID: connectionID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromConnectorID_Success() {
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsDeleteFromPSUIDAndConnectorIDActivity, mock.Anything, psuID, connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromPSUIDAndConnectorIDActivity, mock.Anything, psuID, connectorID).Once().Return(nil)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		FromConnectorID: &DeleteOpenBankingConnectionDataFromConnectorID{
			PSUID:       psuID,
			ConnectorID: connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromPSUID_Success() {
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsDeleteFromPSUIDActivity, mock.Anything, psuID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromPSUIDActivity, mock.Anything, psuID).Once().Return(nil)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		FromPSUID: &DeleteOpenBankingConnectionDataFromPSUID{
			PSUID: psuID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromConnectionID_DeletePayments_Error() {
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsDeleteFromConnectionIDActivity, mock.Anything, psuID, connectorID, connectionID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		FromConnectionID: &DeleteOpenBankingConnectionDataFromConnectionID{
			PSUID:        psuID,
			ConnectorID:  connectorID,
			ConnectionID: connectionID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromConnectionID_DeleteAccounts_Error() {
	connectionID := "test-connection-id"
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsDeleteFromConnectionIDActivity, mock.Anything, psuID, connectorID, connectionID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromConnectionIDActivity, mock.Anything, psuID, connectorID, connectionID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		FromConnectionID: &DeleteOpenBankingConnectionDataFromConnectionID{
			PSUID:        psuID,
			ConnectorID:  connectorID,
			ConnectionID: connectionID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromConnectorID_DeletePayments_Error() {
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsDeleteFromPSUIDAndConnectorIDActivity, mock.Anything, psuID, connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		FromConnectorID: &DeleteOpenBankingConnectionDataFromConnectorID{
			PSUID:       psuID,
			ConnectorID: connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "deleting payments")
}

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromPSUID_DeletePayments_Error() {
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsDeleteFromPSUIDActivity, mock.Anything, psuID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		FromPSUID: &DeleteOpenBankingConnectionDataFromPSUID{
			PSUID: psuID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "deleting payments")
}

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromPSUID_DeleteAccounts_Error() {
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsDeleteFromPSUIDActivity, mock.Anything, psuID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromPSUIDActivity, mock.Anything, psuID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		FromPSUID: &DeleteOpenBankingConnectionDataFromPSUID{
			PSUID: psuID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "deleting accounts")
}
