package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

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

	task = models.Task{
		ID:              models.TaskID{Reference: "test", ConnectorID: connectorID},
		ConnectorID:     &connectorID,
		Status:          models.TASK_STATUS_SUCCEEDED,
		CreatedObjectID: pointer.For("test"),
	}

	balance = models.Balance{
		AccountID: models.AccountID{
			Reference:   "test",
			ConnectorID: connectorID,
		},
		Asset:   "USD/2",
		Balance: big.NewInt(100),
	}

	paymentID = models.PaymentID{
		PaymentReference: models.PaymentReference{
			Reference: "test",
			Type:      models.PAYMENT_TYPE_PAYIN,
		},
		ConnectorID: connectorID,
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

	userPendingDisconnect = models.UserConnectionPendingDisconnect{
		PsuID:        uuid.New(),
		ConnectorID:  connectorID,
		ConnectionID: "test-connection-id",
		At:           time.Now().UTC(),
		Reason:       pointer.For("test-reason"),
	}

	userDisconnected = models.UserDisconnected{
		PsuID:       uuid.New(),
		ConnectorID: connectorID,
		At:          time.Now().UTC(),
		Reason:      pointer.For("test-reason"),
	}

	userConnectionDisconnected = models.UserConnectionDisconnected{
		PsuID:        uuid.New(),
		ConnectorID:  connectorID,
		ConnectionID: "test-connection-id",
		ErrorType:    models.ConnectionDisconnectedErrorTypeUserActionNeeded,
		At:           time.Now().UTC(),
		Reason:       pointer.For("test-reason"),
	}

	userConnectionReconnected = models.UserConnectionReconnected{
		PsuID:        uuid.New(),
		ConnectorID:  connectorID,
		ConnectionID: "test-connection-id",
		At:           time.Now().UTC(),
	}

	userLinkStatus = models.UserLinkSessionFinished{
		PsuID:       uuid.New(),
		ConnectorID: connectorID,
		AttemptID:   uuid.New(),
		Status:      models.OpenBankingConnectionAttemptStatusCompleted,
		Error:       nil,
	}

	userConnectionDataSynced = models.UserConnectionDataSynced{
		PsuID:        uuid.New(),
		ConnectorID:  connectorID,
		ConnectionID: "test-connection-id",
		At:           time.Now().UTC(),
	}
)

func (s *UnitTestSuite) Test_RunSendEvents_EmptyInput_Success() {
	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{})
	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_Any_SendEvents_Error() {
	account.CreatedAt = s.env.Now().UTC()
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(temporal.NewNonRetryableApplicationError("error-test", "STORAGE", errors.New("error-test")))

	// the send events function is called for all data
	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		Account: &account,
	})
	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_RunSendEvents_Account_Success() {
	account.CreatedAt = s.env.Now().UTC()
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.Account, &account)
		s.Equal(req.ConnectorID, &connectorID)
		s.Equal(req.IdempotencyKey, account.IdempotencyKey())
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		Account: &account,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_Task_Success() {
	task.CreatedAt = s.env.Now().UTC()
	task.UpdatedAt = s.env.Now().UTC()
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.Task, &task)
		s.Equal(req.ConnectorID, &connectorID)
		s.Equal(req.IdempotencyKey, task.IdempotencyKey())
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		Task: &task,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_Balance_Success() {
	balance.CreatedAt = s.env.Now().UTC()
	balance.LastUpdatedAt = s.env.Now().UTC()
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.Balance, &balance)
		s.Equal(req.ConnectorID, &connectorID)
		s.Equal(req.IdempotencyKey, balance.IdempotencyKey())
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		Balance: &balance,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_BankAccount_Success() {
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.BankAccount, &s.bankAccount)
		s.Nil(req.ConnectorID)
		s.Equal(req.IdempotencyKey, s.bankAccount.IdempotencyKey())
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		BankAccount: &s.bankAccount,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_PaymentDeleted_Success() {
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.PaymentDeleted, &paymentID)
		s.Equal(req.ConnectorID, &connectorID)
		s.Equal(req.IdempotencyKey, fmt.Sprintf("delete:%s", paymentID.String()))
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		PaymentDeleted: &paymentID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
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
	payment.CreatedAt = s.env.Now().UTC()
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
			Raw:       json.RawMessage(`{"test":"test"}`),
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
			Raw:       json.RawMessage(`{"test":"test"}`),
		},
	}
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.Payment, &activities.SendEventsPayment{
			Payment:    payment,
			Adjustment: payment.Adjustments[0],
		})
		s.Equal(req.ConnectorID, &connectorID)
		s.Equal(req.IdempotencyKey, payment.Adjustments[0].IdempotencyKey())
		return nil
	})

	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.Payment, &activities.SendEventsPayment{
			Payment:    payment,
			Adjustment: payment.Adjustments[1],
		})
		s.Equal(req.ConnectorID, &connectorID)
		s.Equal(req.IdempotencyKey, payment.Adjustments[1].IdempotencyKey())
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		Payment: &payment,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_ConnectorReset_Success() {
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.ConnectorReset, &connectorID)
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		ConnectorReset: &connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_PoolCreation_Success() {
	pool.CreatedAt = s.env.Now().UTC()
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.PoolsCreation, &pool)
		s.Nil(req.ConnectorID)
		s.Equal(req.IdempotencyKey, pool.IdempotencyKey())
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		PoolsCreation: &pool,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_PoolDeletion_Success() {
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.PoolsDeletion, &pool.ID)
		s.Nil(req.ConnectorID)
		s.Equal(req.IdempotencyKey, pool.ID.String())
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		PoolsDeletion: &pool.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_PaymentInitiation_Success() {
	paymentInitiation.CreatedAt = s.env.Now().UTC()
	paymentInitiation.ScheduledAt = s.env.Now().UTC()
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.PaymentInitiation, &paymentInitiation)
		s.Equal(req.ConnectorID, &connectorID)
		s.Equal(req.IdempotencyKey, paymentInitiation.IdempotencyKey())
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		PaymentInitiation: &paymentInitiation,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_PaymentInitiationAdjustment_Success() {
	paymentInitiationAdjustment.CreatedAt = s.env.Now().UTC()
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.PaymentInitiationAdjustment, &paymentInitiationAdjustment)
		s.Equal(req.ConnectorID, &connectorID)
		s.Equal(req.IdempotencyKey, paymentInitiationAdjustment.IdempotencyKey())
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		PaymentInitiationAdjustment: &paymentInitiationAdjustment,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_PaymentInitiationRelatedPayment_Success() {
	paymentInitiationAdjustment.CreatedAt = s.env.Now().UTC()
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.PaymentInitiationRelatedPayment, &paymentInitiationRelatedPayment)
		s.Equal(req.ConnectorID, &connectorID)
		s.Equal(req.IdempotencyKey, paymentInitiationRelatedPayment.IdempotencyKey())
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		PaymentInitiationRelatedPayment: &paymentInitiationRelatedPayment,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_UserPendingDisconnect_Success() {
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.UserPendingDisconnect, &userPendingDisconnect)
		s.Equal(req.ConnectorID, &connectorID)
		s.Equal(req.IdempotencyKey, userPendingDisconnect.IdempotencyKey())
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		UserPendingDisconnect: &userPendingDisconnect,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_UserDisconnected_Success() {
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.UserDisconnected, &userDisconnected)
		s.Equal(req.ConnectorID, &connectorID)
		s.Equal(req.IdempotencyKey, userDisconnected.IdempotencyKey())
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		UserDisconnected: &userDisconnected,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_UserConnectionDisconnected_Success() {
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.UserConnectionDisconnected, &userConnectionDisconnected)
		s.Equal(req.ConnectorID, &connectorID)
		s.Equal(req.IdempotencyKey, userConnectionDisconnected.IdempotencyKey())
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		UserConnectionDisconnected: &userConnectionDisconnected,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_UserConnectionReconnected_Success() {
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.UserConnectionReconnected, &userConnectionReconnected)
		s.Equal(req.ConnectorID, &connectorID)
		s.Equal(req.IdempotencyKey, userConnectionReconnected.IdempotencyKey())
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		UserConnectionReconnected: &userConnectionReconnected,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_UserLinkStatus_Success() {
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.UserLinkStatus, &userLinkStatus)
		s.Equal(req.ConnectorID, &connectorID)
		s.Equal(req.IdempotencyKey, userLinkStatus.IdempotencyKey())
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		UserLinkStatus: &userLinkStatus,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_RunSendEvents_UserConnectionDataSynced_Success() {
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.Equal(req.UserConnectionDataSynced, &userConnectionDataSynced)
		s.Equal(req.ConnectorID, &connectorID)
		s.Equal(req.IdempotencyKey, userConnectionDataSynced.IdempotencyKey())
		return nil
	})

	s.env.ExecuteWorkflow(RunSendEvents, SendEvents{
		UserConnectionDataSynced: &userConnectionDataSynced,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}
