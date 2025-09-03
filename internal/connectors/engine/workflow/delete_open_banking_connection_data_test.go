package workflow

import (
	"errors"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromConnectionID_Success() {
	connectionID := "test-connection-id"
	psuID := uuid.New()

	// Mock payments list with connection ID filter
	s.env.OnActivity(activities.StoragePaymentsListActivity, mock.Anything, mock.Anything).Once().Return(
		&bunpaginate.Cursor[models.Payment]{
			PageSize: 2,
			HasMore:  false,
			Data: []models.Payment{
				{
					ConnectorID: s.connectorID,
					ID:          s.paymentPayoutID,
					Metadata: map[string]string{
						models.ObjectConnectionIDMetadataKey: connectionID,
					},
				},
				{
					ConnectorID: s.connectorID,
					ID: models.PaymentID{
						PaymentReference: models.PaymentReference{
							Reference: "test-2",
							Type:      models.PAYMENT_TYPE_PAYOUT,
						},
						ConnectorID: s.connectorID,
					},
					Metadata: map[string]string{
						models.ObjectConnectionIDMetadataKey: connectionID,
					},
				},
			},
		},
		nil,
	)

	// Mock payment deletions
	s.env.OnActivity(activities.StoragePaymentsDeleteActivity, mock.Anything, s.paymentPayoutID).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentsDeleteActivity, mock.Anything, models.PaymentID{
		PaymentReference: models.PaymentReference{
			Reference: "test-2",
			Type:      models.PAYMENT_TYPE_PAYOUT,
		},
		ConnectorID: s.connectorID,
	}).Once().Return(nil)

	// Mock accounts list with connection ID filter
	s.env.OnActivity(activities.StorageAccountsListActivity, mock.Anything, mock.Anything).Once().Return(
		&bunpaginate.Cursor[models.Account]{
			PageSize: 1,
			HasMore:  false,
			Data: []models.Account{
				{
					ConnectorID: s.connectorID,
					ID:          s.accountID,
					Metadata: map[string]string{
						models.ObjectConnectionIDMetadataKey: connectionID,
					},
				},
			},
		},
		nil,
	)

	// Mock account deletion
	s.env.OnActivity(activities.StorageAccountsDeleteActivity, mock.Anything, s.accountID).Once().Return(nil)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		PSUID: psuID,
		FromConnectionID: &DeleteOpenBankingConnectionDataFromConnectionID{
			ConnectionID: connectionID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromAccountID_Success() {
	accountID := models.AccountID{
		Reference:   "test-account",
		ConnectorID: s.connectorID,
	}

	// Mock payment deletion from account ID
	s.env.OnActivity(activities.StoragePaymentsDeleteFromAccountIDActivity, mock.Anything, accountID).Once().Return(nil)

	// Mock account deletion
	s.env.OnActivity(activities.StorageAccountsDeleteActivity, mock.Anything, accountID).Once().Return(nil)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		PSUID: uuid.New(),
		FromAccountID: &DeleteOpenBankingConnectionDataFromAccountID{
			AccountID: accountID,
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

	// Mock payments list with connector ID filter
	s.env.OnActivity(activities.StoragePaymentsListActivity, mock.Anything, mock.Anything).Once().Return(
		&bunpaginate.Cursor[models.Payment]{
			PageSize: 1,
			HasMore:  false,
			Data: []models.Payment{
				{
					ConnectorID: connectorID,
					ID:          s.paymentPayoutID,
					Metadata: map[string]string{
						models.ObjectPSUIDMetadataKey: psuID.String(),
						"connector_id":                connectorID.String(),
					},
				},
			},
		},
		nil,
	)

	// Mock payment deletion
	s.env.OnActivity(activities.StoragePaymentsDeleteActivity, mock.Anything, s.paymentPayoutID).Once().Return(nil)

	// Mock accounts list with connector ID filter
	s.env.OnActivity(activities.StorageAccountsListActivity, mock.Anything, mock.Anything).Once().Return(
		&bunpaginate.Cursor[models.Account]{
			PageSize: 1,
			HasMore:  false,
			Data: []models.Account{
				{
					ConnectorID: connectorID,
					ID:          s.accountID,
					Metadata: map[string]string{
						models.ObjectPSUIDMetadataKey: psuID.String(),
						"connector_id":                connectorID.String(),
					},
				},
			},
		},
		nil,
	)

	// Mock account deletion
	s.env.OnActivity(activities.StorageAccountsDeleteActivity, mock.Anything, s.accountID).Once().Return(nil)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		PSUID: psuID,
		FromConnectorID: &DeleteOpenBankingConnectionDataFromConnectorID{
			ConnectorID: connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromPSUID_Success() {
	psuID := uuid.New()

	// Mock payments list with PSU ID filter
	s.env.OnActivity(activities.StoragePaymentsListActivity, mock.Anything, mock.Anything).Once().Return(
		&bunpaginate.Cursor[models.Payment]{
			PageSize: 1,
			HasMore:  false,
			Data: []models.Payment{
				{
					ConnectorID: s.connectorID,
					ID:          s.paymentPayoutID,
					Metadata: map[string]string{
						models.ObjectPSUIDMetadataKey: psuID.String(),
					},
				},
			},
		},
		nil,
	)

	// Mock payment deletion
	s.env.OnActivity(activities.StoragePaymentsDeleteActivity, mock.Anything, s.paymentPayoutID).Once().Return(nil)

	// Mock accounts list with PSU ID filter
	s.env.OnActivity(activities.StorageAccountsListActivity, mock.Anything, mock.Anything).Once().Return(
		&bunpaginate.Cursor[models.Account]{
			PageSize: 1,
			HasMore:  false,
			Data: []models.Account{
				{
					ConnectorID: s.connectorID,
					ID:          s.accountID,
					Metadata: map[string]string{
						models.ObjectPSUIDMetadataKey: psuID.String(),
					},
				},
			},
		},
		nil,
	)

	// Mock account deletion
	s.env.OnActivity(activities.StorageAccountsDeleteActivity, mock.Anything, s.accountID).Once().Return(nil)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		PSUID: psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromAccountID_StoragePaymentsDeleteFromAccountID_Error() {
	accountID := models.AccountID{
		Reference:   "test-account",
		ConnectorID: s.connectorID,
	}

	s.env.OnActivity(activities.StoragePaymentsDeleteFromAccountIDActivity, mock.Anything, accountID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		PSUID: uuid.New(),
		FromAccountID: &DeleteOpenBankingConnectionDataFromAccountID{
			AccountID: accountID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "deleting payments from account ID")
}

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromAccountID_StorageAccountsDelete_Error() {
	accountID := models.AccountID{
		Reference:   "test-account",
		ConnectorID: s.connectorID,
	}

	s.env.OnActivity(activities.StoragePaymentsDeleteFromAccountIDActivity, mock.Anything, accountID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteActivity, mock.Anything, accountID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		PSUID: uuid.New(),
		FromAccountID: &DeleteOpenBankingConnectionDataFromAccountID{
			AccountID: accountID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "deleting account")
}

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromConnectionID_StoragePaymentsList_Error() {
	connectionID := "test-connection-id"
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsListActivity, mock.Anything, mock.Anything).Once().Return(
		(*bunpaginate.Cursor[models.Payment])(nil), temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		PSUID: psuID,
		FromConnectionID: &DeleteOpenBankingConnectionDataFromConnectionID{
			ConnectionID: connectionID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "deleting payments")
}

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromConnectionID_StorageAccountsList_Error() {
	connectionID := "test-connection-id"
	psuID := uuid.New()

	// Mock successful payments deletion
	s.env.OnActivity(activities.StoragePaymentsListActivity, mock.Anything, mock.Anything).Once().Return(
		&bunpaginate.Cursor[models.Payment]{
			PageSize: 0,
			HasMore:  false,
			Data:     []models.Payment{},
		},
		nil,
	)

	s.env.OnActivity(activities.StorageAccountsListActivity, mock.Anything, mock.Anything).Once().Return(
		(*bunpaginate.Cursor[models.Account])(nil), temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		PSUID: psuID,
		FromConnectionID: &DeleteOpenBankingConnectionDataFromConnectionID{
			ConnectionID: connectionID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "deleting accounts")
}

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromConnectorID_StoragePaymentsList_Error() {
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsListActivity, mock.Anything, mock.Anything).Once().Return(
		(*bunpaginate.Cursor[models.Payment])(nil), temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		PSUID: psuID,
		FromConnectorID: &DeleteOpenBankingConnectionDataFromConnectorID{
			ConnectorID: connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "deleting payments")
}

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromPSUID_StoragePaymentsList_Error() {
	psuID := uuid.New()

	s.env.OnActivity(activities.StoragePaymentsListActivity, mock.Anything, mock.Anything).Once().Return(
		(*bunpaginate.Cursor[models.Payment])(nil), temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		PSUID: psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "deleting payments")
}

func (s *UnitTestSuite) Test_DeleteOpenBankingConnectionData_FromPSUID_StorageAccountsList_Error() {
	psuID := uuid.New()

	// Mock successful payments deletion
	s.env.OnActivity(activities.StoragePaymentsListActivity, mock.Anything, mock.Anything).Once().Return(
		&bunpaginate.Cursor[models.Payment]{
			PageSize: 0,
			HasMore:  false,
			Data:     []models.Payment{},
		},
		nil,
	)

	s.env.OnActivity(activities.StorageAccountsListActivity, mock.Anything, mock.Anything).Once().Return(
		(*bunpaginate.Cursor[models.Account])(nil), temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeleteOpenBankingConnectionData, DeleteOpenBankingConnectionData{
		PSUID: psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "deleting accounts")
}
