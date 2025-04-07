package workflow

import (
	"math/big"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

var (
	connectorID = models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	account = models.Account{
		ID: models.AccountID{
			Reference:   "test",
			ConnectorID: connectorID,
		},
		ConnectorID: connectorID,
		Reference:   "test",
		Type:        "INTERNAL",
		Raw:         []byte(`{"test":"test"}`),
	}

	balance = models.Balance{
		AccountID: models.AccountID{
			Reference:   "test",
			ConnectorID: connectorID,
		},
		Asset:   "USD/2",
		Balance: big.NewInt(100),
	}

	payment = models.Payment{
		ID: models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: "test",
				Type:      models.PAYMENT_TYPE_PAYIN,
			},
			ConnectorID: connectorID,
		},
		ConnectorID: connectorID,
		Reference:   "test",
		Type:        models.PAYMENT_TYPE_PAYIN,
		Amount:      big.NewInt(100),
		Asset:       "USD/2",
		Status:      models.PAYMENT_STATUS_SUCCEEDED,
	}

	pool = models.Pool{
		ID:   uuid.New(),
		Name: "test",
		PoolAccounts: []models.AccountID{
			account.ID,
		},
	}

	paymentInitiation = models.PaymentInitiation{
		ID: models.PaymentInitiationID{
			Reference:   "test",
			ConnectorID: connectorID,
		},
		ConnectorID: connectorID,
		Reference:   "test",
		Description: "test",
		Type:        models.PAYMENT_INITIATION_TYPE_PAYOUT,
		Amount:      big.NewInt(100),
		Asset:       "USD/2",
	}

	paymentInitiationAdjustment = models.PaymentInitiationAdjustment{
		ID: models.PaymentInitiationAdjustmentID{
			PaymentInitiationID: models.PaymentInitiationID{
				Reference:   "test",
				ConnectorID: connectorID,
			},
			Status: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
		},
		Status: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
		Amount: big.NewInt(100),
		Asset:  pointer.For("USD/2"),
	}

	paymentInitiationRelatedPayment = models.PaymentInitiationRelatedPayments{
		PaymentInitiationID: models.PaymentInitiationID{
			Reference:   "test",
			ConnectorID: connectorID,
		},
		PaymentID: payment.ID,
	}
)

func (s *UnitTestSuite) Test_RunSendEvents_EmptyInput_Success() {
	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{})
	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_Any_EventsSentGet_Error() {
	account.CreatedAt = s.env.Now()
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: account.IdempotencyKey(),
		ConnectorID:         &connectorID,
	}).Return(true, temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test")))

	// the send events function is called for all data
	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		Account: &account,
	})
	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "test")
}

func (s *UnitTestSuite) Test_RunSendEvents_Any_EventsSentStore_Error() {
	account.CreatedAt = s.env.Now()
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: account.IdempotencyKey(),
		ConnectorID:         &connectorID,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendAccountActivity, mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity(activities.StorageEventsSentStoreActivity, mock.Anything, mock.Anything).Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test")),
	)

	// the send events function is called for all data
	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		Account: &account,
	})
	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "test")
}

func (s *UnitTestSuite) Test_RunSendEvents_Account_Event_Exists_Success() {
	account.CreatedAt = s.env.Now()
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: account.IdempotencyKey(),
		ConnectorID:         &connectorID,
	}).Return(true, nil)

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		Account: &account,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_Account_Success() {
	account.CreatedAt = s.env.Now()
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: account.IdempotencyKey(),
		ConnectorID:         &connectorID,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendAccountActivity, mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity(activities.StorageEventsSentStoreActivity, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		Account: &account,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_Account_Error() {
	account.CreatedAt = s.env.Now()
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: account.IdempotencyKey(),
		ConnectorID:         &connectorID,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendAccountActivity, mock.Anything, mock.Anything).Return(errors.New("test"))

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		Account: &account,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "test")
}

func (s *UnitTestSuite) Test_RunSendEvents_Balance_Success() {
	balance.CreatedAt = s.env.Now()
	balance.LastUpdatedAt = s.env.Now()
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: balance.IdempotencyKey(),
		ConnectorID:         &connectorID,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendBalanceActivity, mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity(activities.StorageEventsSentStoreActivity, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		Balance: &balance,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_Balance_Error() {
	balance.CreatedAt = s.env.Now()
	balance.LastUpdatedAt = s.env.Now()
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: balance.IdempotencyKey(),
		ConnectorID:         &connectorID,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendBalanceActivity, mock.Anything, mock.Anything).Return(errors.New("test"))

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		Balance: &balance,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "test")
}

func (s *UnitTestSuite) Test_RunSendEvents_BankAccount_Success() {
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: s.bankAccount.IdempotencyKey(),
		ConnectorID:         nil,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendBankAccountActivity, mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity(activities.StorageEventsSentStoreActivity, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		BankAccount: &s.bankAccount,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_BankAccount_Error() {
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: s.bankAccount.IdempotencyKey(),
		ConnectorID:         nil,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendBankAccountActivity, mock.Anything, mock.Anything).Return(errors.New("test"))

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		BankAccount: &s.bankAccount,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "test")
}

func (s *UnitTestSuite) Test_RunSendEvents_Payment_NoAdjustments_Success() {
	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		Payment: &payment,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_Payment_WithAdjustments_Success() {
	payment.CreatedAt = s.env.Now()
	payment.Adjustments = []models.PaymentAdjustment{
		{
			ID: models.PaymentAdjustmentID{
				PaymentID: payment.ID,
				Reference: "test1",
				Status:    models.PAYMENT_STATUS_PENDING,
			},
			Reference: "test1",
			Status:    models.PAYMENT_STATUS_PENDING,
			Amount:    big.NewInt(100),
			Asset:     pointer.For("USD/2"),
		},
		{
			ID: models.PaymentAdjustmentID{
				PaymentID: payment.ID,
				Reference: "test1",
				Status:    models.PAYMENT_STATUS_SUCCEEDED,
			},
			Reference: "test1",
			Status:    models.PAYMENT_STATUS_SUCCEEDED,
			Amount:    big.NewInt(100),
			Asset:     pointer.For("USD/2"),
		},
	}
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: payment.Adjustments[0].IdempotencyKey(),
		ConnectorID:         &connectorID,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendPaymentActivity, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity(activities.StorageEventsSentStoreActivity, mock.Anything, mock.Anything).Return(nil)

	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: payment.Adjustments[1].IdempotencyKey(),
		ConnectorID:         &connectorID,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendPaymentActivity, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity(activities.StorageEventsSentStoreActivity, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		Payment: &payment,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_Payment_WithAdjustments_Error() {
	payment.CreatedAt = s.env.Now()
	payment.Adjustments = []models.PaymentAdjustment{
		{
			ID: models.PaymentAdjustmentID{
				PaymentID: payment.ID,
				Reference: "test1",
				Status:    models.PAYMENT_STATUS_PENDING,
			},
			Reference: "test1",
			Status:    models.PAYMENT_STATUS_PENDING,
			Amount:    big.NewInt(100),
			Asset:     pointer.For("USD/2"),
		},
		{
			ID: models.PaymentAdjustmentID{
				PaymentID: payment.ID,
				Reference: "test1",
				Status:    models.PAYMENT_STATUS_SUCCEEDED,
			},
			Reference: "test1",
			Status:    models.PAYMENT_STATUS_SUCCEEDED,
			Amount:    big.NewInt(100),
			Asset:     pointer.For("USD/2"),
		},
	}
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: payment.Adjustments[0].IdempotencyKey(),
		ConnectorID:         &connectorID,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendPaymentActivity, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("test"))

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		Payment: &payment,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "test")
}

func (s *UnitTestSuite) Test_RunSendEvents_ConnectorReset_Success() {
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, mock.Anything).Return(false, nil)
	s.env.OnActivity(activities.EventsSendConnectorResetActivity, mock.Anything, connectorID, mock.Anything).Return(nil)
	s.env.OnActivity(activities.StorageEventsSentStoreActivity, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		ConnectorReset: &connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_ConnectorReset_Error() {
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, mock.Anything).Return(false, nil)
	s.env.OnActivity(activities.EventsSendConnectorResetActivity, mock.Anything, connectorID, mock.Anything).Return(errors.New("test"))

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		ConnectorReset: &connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "test")
}

func (s *UnitTestSuite) Test_RunSendEvents_PoolCreation_Success() {
	pool.CreatedAt = s.env.Now()
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: pool.IdempotencyKey(),
		ConnectorID:         nil,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendPoolCreationActivity, mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity(activities.StorageEventsSentStoreActivity, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		PoolsCreation: &pool,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_PoolCreation_Error() {
	pool.CreatedAt = s.env.Now()
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: pool.IdempotencyKey(),
		ConnectorID:         nil,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendPoolCreationActivity, mock.Anything, mock.Anything).Return(errors.New("test"))

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		PoolsCreation: &pool,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "test")
}

func (s *UnitTestSuite) Test_RunSendEvents_PoolDeletion_Success() {
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: pool.ID.String(),
		ConnectorID:         nil,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendPoolDeletionActivity, mock.Anything, pool.ID).Return(nil)
	s.env.OnActivity(activities.StorageEventsSentStoreActivity, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		PoolsDeletion: &pool.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_Pool_Error() {
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: pool.ID.String(),
		ConnectorID:         nil,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendPoolDeletionActivity, mock.Anything, pool.ID).Return(errors.New("test"))

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		PoolsDeletion: &pool.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "test")
}

func (s *UnitTestSuite) Test_RunSendEvents_PaymentInitiation_Success() {
	paymentInitiation.CreatedAt = s.env.Now()
	paymentInitiation.ScheduledAt = s.env.Now()
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: paymentInitiation.IdempotencyKey(),
		ConnectorID:         &connectorID,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendPaymentInitiationActivity, mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity(activities.StorageEventsSentStoreActivity, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		PaymentInitiation: &paymentInitiation,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_PaymentInitiation_Error() {
	paymentInitiation.CreatedAt = s.env.Now()
	paymentInitiation.ScheduledAt = s.env.Now()
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: paymentInitiation.IdempotencyKey(),
		ConnectorID:         &connectorID,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendPaymentInitiationActivity, mock.Anything, mock.Anything).Return(errors.New("test"))

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		PaymentInitiation: &paymentInitiation,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "test")
}

func (s *UnitTestSuite) Test_RunSendEvents_PaymentInitiationAdjustment_Success() {
	paymentInitiationAdjustment.CreatedAt = s.env.Now()
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: paymentInitiationAdjustment.IdempotencyKey(),
		ConnectorID:         &connectorID,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendPaymentInitiationAdjustmentActivity, mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity(activities.StorageEventsSentStoreActivity, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		PaymentInitiationAdjustment: &paymentInitiationAdjustment,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_PaymentInitiationAdjustment_Error() {
	paymentInitiationAdjustment.CreatedAt = s.env.Now()
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: paymentInitiationAdjustment.IdempotencyKey(),
		ConnectorID:         &connectorID,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendPaymentInitiationAdjustmentActivity, mock.Anything, mock.Anything).Return(errors.New("test"))

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		PaymentInitiationAdjustment: &paymentInitiationAdjustment,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "test")
}

func (s *UnitTestSuite) Test_RunSendEvents_PaymentInitiationRelatedPayment_Success() {
	paymentInitiationAdjustment.CreatedAt = s.env.Now()
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: paymentInitiationRelatedPayment.IdempotencyKey(),
		ConnectorID:         &connectorID,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendPaymentInitiationRelatedPaymentActivity, mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity(activities.StorageEventsSentStoreActivity, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		PaymentInitiationRelatedPayment: &paymentInitiationRelatedPayment,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_PaymentInitiationRelatedPayment_Error() {
	paymentInitiationAdjustment.CreatedAt = s.env.Now()
	s.env.OnActivity(activities.StorageEventsSentGetActivity, mock.Anything, models.EventID{
		EventIdempotencyKey: paymentInitiationRelatedPayment.IdempotencyKey(),
		ConnectorID:         &connectorID,
	}).Return(false, nil)
	s.env.OnActivity(activities.EventsSendPaymentInitiationRelatedPaymentActivity, mock.Anything, mock.Anything).Return(errors.New("test"))

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		PaymentInitiationRelatedPayment: &paymentInitiationRelatedPayment,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "test")
}
