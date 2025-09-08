package workflow

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_DeleteBankBridgeConnectionData_FromConnectionID_Success() {
	connectionID := "test-connection-id"
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsDeleteFromConnectionIDActivity, mock.Anything, psuID, connectionID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromConnectionIDActivity, mock.Anything, psuID, connectionID).Once().Return(nil)

	s.env.ExecuteWorkflow(RunDeleteBankBridgeConnectionData, DeleteBankBridgeConnectionData{
		PSUID: psuID,
		FromConnectionID: &DeleteBankBridgeConnectionDataFromConnectionID{
			ConnectionID: connectionID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_DeleteBankBridgeConnectionData_FromAccountID_Success() {
	accountID := models.AccountID{
		Reference:   "test-account",
		ConnectorID: s.connectorID,
	}

	// Mock payment deletion from account ID
	s.env.OnActivity(activities.StoragePaymentsDeleteFromAccountIDActivity, mock.Anything, accountID).Once().Return(nil)

	// Mock account deletion
	s.env.OnActivity(activities.StorageAccountsDeleteActivity, mock.Anything, accountID).Once().Return(nil)

	s.env.ExecuteWorkflow(RunDeleteBankBridgeConnectionData, DeleteBankBridgeConnectionData{
		PSUID: uuid.New(),
		FromAccountID: &DeleteBankBridgeConnectionDataFromAccountID{
			AccountID: accountID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_DeleteBankBridgeConnectionData_FromConnectorID_Success() {
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsDeleteFromPSUIDAndConnectorIDActivity, mock.Anything, psuID, connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromPSUIDAndConnectorIDActivity, mock.Anything, psuID, connectorID).Once().Return(nil)

	s.env.ExecuteWorkflow(RunDeleteBankBridgeConnectionData, DeleteBankBridgeConnectionData{
		PSUID: psuID,
		FromConnectorID: &DeleteBankBridgeConnectionDataFromConnectorID{
			ConnectorID: connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_DeleteBankBridgeConnectionData_FromPSUID_Success() {
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsDeleteFromPSUIDActivity, mock.Anything, psuID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromPSUIDActivity, mock.Anything, psuID).Once().Return(nil)

	s.env.ExecuteWorkflow(RunDeleteBankBridgeConnectionData, DeleteBankBridgeConnectionData{
		PSUID: psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_DeleteBankBridgeConnectionData_FromAccountID_StoragePaymentsDeleteFromAccountID_Error() {
	accountID := models.AccountID{
		Reference:   "test-account",
		ConnectorID: s.connectorID,
	}

	s.env.OnActivity(activities.StoragePaymentsDeleteFromAccountIDActivity, mock.Anything, accountID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteBankBridgeConnectionData, DeleteBankBridgeConnectionData{
		PSUID: uuid.New(),
		FromAccountID: &DeleteBankBridgeConnectionDataFromAccountID{
			AccountID: accountID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "deleting payments from account ID")
}

func (s *UnitTestSuite) Test_DeleteBankBridgeConnectionData_FromAccountID_StorageAccountsDelete_Error() {
	accountID := models.AccountID{
		Reference:   "test-account",
		ConnectorID: s.connectorID,
	}

	s.env.OnActivity(activities.StoragePaymentsDeleteFromAccountIDActivity, mock.Anything, accountID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteActivity, mock.Anything, accountID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteBankBridgeConnectionData, DeleteBankBridgeConnectionData{
		PSUID: uuid.New(),
		FromAccountID: &DeleteBankBridgeConnectionDataFromAccountID{
			AccountID: accountID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "deleting account")
}

func (s *UnitTestSuite) Test_DeleteBankBridgeConnectionData_FromConnectionID_DeletePayments_Error() {
	connectionID := "test-connection-id"
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsDeleteFromConnectionIDActivity, mock.Anything, psuID, connectionID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteBankBridgeConnectionData, DeleteBankBridgeConnectionData{
		PSUID: psuID,
		FromConnectionID: &DeleteBankBridgeConnectionDataFromConnectionID{
			ConnectionID: connectionID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeleteBankBridgeConnectionData_FromConnectionID_DeleteAccounts_Error() {
	connectionID := "test-connection-id"
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsDeleteFromConnectionIDActivity, mock.Anything, psuID, connectionID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromConnectionIDActivity, mock.Anything, psuID, connectionID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteBankBridgeConnectionData, DeleteBankBridgeConnectionData{
		PSUID: psuID,
		FromConnectionID: &DeleteBankBridgeConnectionDataFromConnectionID{
			ConnectionID: connectionID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeleteBankBridgeConnectionData_FromConnectorID_DeletePayments_Error() {
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsDeleteFromPSUIDAndConnectorIDActivity, mock.Anything, psuID, connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteBankBridgeConnectionData, DeleteBankBridgeConnectionData{
		PSUID: psuID,
		FromConnectorID: &DeleteBankBridgeConnectionDataFromConnectorID{
			ConnectorID: connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "deleting payments")
}

func (s *UnitTestSuite) Test_DeleteBankBridgeConnectionData_FromPSUID_DeletePayments_Error() {
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsDeleteFromPSUIDActivity, mock.Anything, psuID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteBankBridgeConnectionData, DeleteBankBridgeConnectionData{
		PSUID: psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "deleting payments")
}

func (s *UnitTestSuite) Test_DeleteBankBridgeConnectionData_FromPSUID_DeleteAccounts_Error() {
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsDeleteFromPSUIDActivity, mock.Anything, psuID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromPSUIDActivity, mock.Anything, psuID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteBankBridgeConnectionData, DeleteBankBridgeConnectionData{
		PSUID: psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "deleting accounts")
}
