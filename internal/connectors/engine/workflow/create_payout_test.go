package workflow

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func (s *UnitTestSuite) Test_CreatePayout_WithPayment_Success() {
	s.env.OnActivity(activities.StoragePaymentInitiationsGetActivity, mock.Anything, s.paymentInitiationID).Once().Return(&s.paymentInitiationPayout, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.SourceAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.DestinationAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, adj models.PaymentInitiationAdjustment) error {
		s.Equal(s.paymentInitiationID, adj.ID.PaymentInitiationID)
		s.Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING, adj.Status)
		s.Equal(big.NewInt(100), adj.Amount)
		s.NotNil(adj.Asset)
		s.Equal("USD/2", *adj.Asset)
		s.Nil(adj.Error)
		return nil
	})
	s.env.OnActivity(activities.PluginCreatePayoutActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CreatePayoutRequest) (*models.CreatePayoutResponse, error) {
		s.Equal(s.connectorID, req.ConnectorID)
		s.Equal(s.paymentInitiationID.Reference, req.Req.PaymentInitiation.Reference)
		return &models.CreatePayoutResponse{
			Payment: &s.pspPayment,
		}, nil
	})
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, payments []models.Payment) error {
		s.Equal(1, len(payments))
		s.Equal(s.paymentPayoutID, payments[0].ID)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Payment)
		s.Nil(req.Account)
		s.Nil(req.Balance)
		s.Nil(req.BankAccount)
		s.Nil(req.ConnectorReset)
		s.Nil(req.PoolsCreation)
		s.Nil(req.PoolsDeletion)
		return nil
	})
	s.env.OnActivity(activities.StoragePaymentInitiationsRelatedPaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, relatedPayment activities.RelatedPayment) error {
		s.Equal(s.paymentInitiationID, relatedPayment.PiID)
		s.Equal(s.paymentPayoutID, relatedPayment.PID)
		return nil
	})
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, adj models.PaymentInitiationAdjustment) error {
		s.Equal(s.paymentInitiationID, adj.ID.PaymentInitiationID)
		s.Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED, adj.Status)
		s.Equal(big.NewInt(100), adj.Amount)
		s.NotNil(adj.Asset)
		s.Equal("USD/2", *adj.Asset)
		s.Nil(adj.Error)
		return nil
	})
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_SUCCEEDED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreatePayout, CreatePayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreatePayout_WithScheduledAt_WithPayment_Success() {
	paymentInitiationPayout := s.paymentInitiationPayout
	paymentInitiationPayout.ScheduledAt = s.env.Now().Add(1 * time.Hour)
	s.env.OnActivity(activities.StoragePaymentInitiationsGetActivity, mock.Anything, s.paymentInitiationID).Once().Return(&paymentInitiationPayout, nil)

	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, adj models.PaymentInitiationAdjustment) error {
		s.Equal(s.paymentInitiationID, adj.ID.PaymentInitiationID)
		s.Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_SCHEDULED_FOR_PROCESSING, adj.Status)
		s.Equal(big.NewInt(100), adj.Amount)
		s.NotNil(adj.Asset)
		s.Equal("USD/2", *adj.Asset)
		s.Nil(adj.Error)
		return nil
	})

	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.SourceAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.DestinationAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, adj models.PaymentInitiationAdjustment) error {
		s.Equal(s.paymentInitiationID, adj.ID.PaymentInitiationID)
		s.Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING, adj.Status)
		s.Equal(big.NewInt(100), adj.Amount)
		s.NotNil(adj.Asset)
		s.Equal("USD/2", *adj.Asset)
		s.Nil(adj.Error)
		return nil
	})
	s.env.OnActivity(activities.PluginCreatePayoutActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CreatePayoutRequest) (*models.CreatePayoutResponse, error) {
		s.Equal(s.connectorID, req.ConnectorID)
		s.Equal(s.paymentInitiationID.Reference, req.Req.PaymentInitiation.Reference)
		return &models.CreatePayoutResponse{
			Payment: &s.pspPayment,
		}, nil
	})
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, payments []models.Payment) error {
		s.Equal(1, len(payments))
		s.Equal(s.paymentPayoutID, payments[0].ID)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Payment)
		s.Nil(req.Account)
		s.Nil(req.Balance)
		s.Nil(req.BankAccount)
		s.Nil(req.ConnectorReset)
		s.Nil(req.PoolsCreation)
		s.Nil(req.PoolsDeletion)
		return nil
	})
	s.env.OnActivity(activities.StoragePaymentInitiationsRelatedPaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, relatedPayment activities.RelatedPayment) error {
		s.Equal(s.paymentInitiationID, relatedPayment.PiID)
		s.Equal(s.paymentPayoutID, relatedPayment.PID)
		return nil
	})
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, adj models.PaymentInitiationAdjustment) error {
		s.Equal(s.paymentInitiationID, adj.ID.PaymentInitiationID)
		s.Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED, adj.Status)
		s.Equal(big.NewInt(100), adj.Amount)
		s.NotNil(adj.Asset)
		s.Equal("USD/2", *adj.Asset)
		s.Nil(adj.Error)
		return nil
	})
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_SUCCEEDED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreatePayout, CreatePayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreatePayout_WithPollingPayment_Success() {
	s.env.OnActivity(activities.StoragePaymentInitiationsGetActivity, mock.Anything, s.paymentInitiationID).Once().Return(&s.paymentInitiationPayout, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.SourceAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.DestinationAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, adj models.PaymentInitiationAdjustment) error {
		s.Equal(s.paymentInitiationID, adj.ID.PaymentInitiationID)
		s.Equal(big.NewInt(100), adj.Amount)
		s.NotNil(adj.Asset)
		s.Equal("USD/2", *adj.Asset)
		s.Nil(adj.Error)
		return nil
	})
	s.env.OnActivity(activities.PluginCreatePayoutActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CreatePayoutRequest) (*models.CreatePayoutResponse, error) {
		s.Equal(s.connectorID, req.ConnectorID)
		s.Equal(s.paymentInitiationID.Reference, req.Req.PaymentInitiation.Reference)
		return &models.CreatePayoutResponse{
			PollingPayoutID: pointer.For("test"),
		}, nil
	})
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, schedule models.Schedule) error {
		s.Contains(schedule.ID, "polling-payout")
		s.Equal(s.connectorID, schedule.ConnectorID)
		return nil
	})
	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, options activities.ScheduleCreateOptions) error {
		s.Contains(options.ScheduleID, "polling-payout")
		s.Equal(RunPollPayout, options.Action.Workflow)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreatePayout, CreatePayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreatePayout_StoragePaymentInitiationsGet_Error() {
	s.env.OnActivity(activities.StoragePaymentInitiationsGetActivity, mock.Anything, s.paymentInitiationID).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreatePayout, CreatePayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreatePayout_StorageAccountsGet_Error() {
	s.env.OnActivity(activities.StoragePaymentInitiationsGetActivity, mock.Anything, s.paymentInitiationID).Once().Return(&s.paymentInitiationPayout, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.SourceAccountID).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreatePayout, CreatePayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreatePayout_StoragePaymentInitiationsAdjustmentsStore_Error() {
	s.env.OnActivity(activities.StoragePaymentInitiationsGetActivity, mock.Anything, s.paymentInitiationID).Once().Return(&s.paymentInitiationPayout, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.SourceAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.DestinationAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreatePayout, CreatePayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreatePayout_PluginCreatePayout_Error() {
	s.env.OnActivity(activities.StoragePaymentInitiationsGetActivity, mock.Anything, s.paymentInitiationID).Once().Return(&s.paymentInitiationPayout, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.SourceAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.DestinationAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, adj models.PaymentInitiationAdjustment) error {
		s.Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING, adj.Status)
		return nil
	})
	s.env.OnActivity(activities.PluginCreatePayoutActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "PLUGIN", errors.New("test")),
	)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, adj models.PaymentInitiationAdjustment) error {
		s.Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED, adj.Status)
		return nil
	})
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreatePayout, CreatePayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreatePayout_StoragePaymentsStore_Error() {
	s.env.OnActivity(activities.StoragePaymentInitiationsGetActivity, mock.Anything, s.paymentInitiationID).Once().Return(&s.paymentInitiationPayout, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.SourceAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.DestinationAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginCreatePayoutActivity, mock.Anything, mock.Anything).Once().Return(&models.CreatePayoutResponse{
		Payment: &s.pspPayment,
	}, nil)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreatePayout, CreatePayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreatePayout_RunSendEvents_Error() {
	s.env.OnActivity(activities.StoragePaymentInitiationsGetActivity, mock.Anything, s.paymentInitiationID).Once().Return(&s.paymentInitiationPayout, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.SourceAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.DestinationAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginCreatePayoutActivity, mock.Anything, mock.Anything).Once().Return(&models.CreatePayoutResponse{
		Payment: &s.pspPayment,
	}, nil)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(temporal.NewNonRetryableApplicationError("test", "WORKFLOW", errors.New("test")))
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreatePayout, CreatePayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreatePayoutStoragePaymentInitiationsRelatedPaymentsStore_Error() {
	s.env.OnActivity(activities.StoragePaymentInitiationsGetActivity, mock.Anything, s.paymentInitiationID).Once().Return(&s.paymentInitiationPayout, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.SourceAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.DestinationAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginCreatePayoutActivity, mock.Anything, mock.Anything).Once().Return(&models.CreatePayoutResponse{
		Payment: &s.pspPayment,
	}, nil)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsRelatedPaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreatePayout, CreatePayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreatePayout_StorageSchedulesStore_Error() {
	s.env.OnActivity(activities.StoragePaymentInitiationsGetActivity, mock.Anything, s.paymentInitiationID).Once().Return(&s.paymentInitiationPayout, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.SourceAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.DestinationAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginCreatePayoutActivity, mock.Anything, mock.Anything).Once().Return(&models.CreatePayoutResponse{
		PollingPayoutID: pointer.For("test"),
	}, nil)
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreatePayout, CreatePayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreatePayout_TemporalScheduleCreate_Error() {
	s.env.OnActivity(activities.StoragePaymentInitiationsGetActivity, mock.Anything, s.paymentInitiationID).Once().Return(&s.paymentInitiationPayout, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.SourceAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, *s.paymentInitiationPayout.DestinationAccountID).Once().Return(&s.account, nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginCreatePayoutActivity, mock.Anything, mock.Anything).Once().Return(&models.CreatePayoutResponse{
		PollingPayoutID: pointer.For("test"),
	}, nil)
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "SCHEDULE", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreatePayout, CreatePayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreatePayout_StorageTasksStoreActivity_Error() {
	s.env.OnActivity(activities.StoragePaymentInitiationsGetActivity, mock.Anything, s.paymentInitiationID).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test"))
	})

	s.env.ExecuteWorkflow(RunCreatePayout, CreatePayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}
