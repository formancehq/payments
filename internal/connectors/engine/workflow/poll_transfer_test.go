package workflow

import (
	"context"
	"fmt"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func (s *UnitTestSuite) Test_PollTransfer_WithPayment_Success() {
	s.env.OnActivity(activities.PluginPollTransferStatusActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.PollTransferStatusRequest) (*models.PollTransferStatusResponse, error) {
		s.Equal(s.connectorID, req.ConnectorID)
		s.Equal("test-transfer", req.Req.TransferID)
		return &models.PollTransferStatusResponse{
			Payment: &s.pspPayment,
		}, nil
	})
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, payments []models.Payment) error {
		s.Len(payments, 1)
		s.Equal(s.paymentPayoutID, payments[0].ID)
		return nil
	})
	s.env.OnActivity(activities.StoragePaymentInitiationsRelatedPaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, relatedPayment activities.RelatedPayment) error {
		s.Equal(s.paymentInitiationID, relatedPayment.PiID)
		s.Equal(s.paymentPayoutID, relatedPayment.PID)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, sendEvents SendEvents) error {
		s.Nil(sendEvents.Balance)
		s.Nil(sendEvents.Account)
		s.Nil(sendEvents.ConnectorReset)
		s.NotNil(sendEvents.Payment)
		s.NotNil(sendEvents.SendEventPaymentInitiationRelatedPayment)
		s.Nil(sendEvents.PoolsCreation)
		s.Nil(sendEvents.PoolsDeletion)
		s.Nil(sendEvents.BankAccount)
		return nil
	})
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, adj models.PaymentInitiationAdjustment) error {
		s.Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED, adj.Status)
		s.Equal(s.paymentInitiationID, adj.ID.PaymentInitiationID)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.SendEventPaymentInitiationAdjustment)
		return nil
	})
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, "test-schedule").Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteActivity, mock.Anything, "test-schedule").Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_SUCCEEDED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunPollTransfer, PollTransfer{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:       s.connectorID,
		PaymentInitiation: &s.paymentInitiationTransfer,
		TransferID:        "test-transfer",
		ScheduleID:        "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollTransfer_WithoutPaymentAndError_Success() {
	s.env.OnActivity(activities.PluginPollTransferStatusActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.PollTransferStatusRequest) (*models.PollTransferStatusResponse, error) {
		s.Equal(s.connectorID, req.ConnectorID)
		s.Equal("test-transfer", req.Req.TransferID)
		return &models.PollTransferStatusResponse{}, nil
	})

	s.env.ExecuteWorkflow(RunPollTransfer, PollTransfer{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:       s.connectorID,
		PaymentInitiation: &s.paymentInitiationTransfer,
		TransferID:        "test-transfer",
		ScheduleID:        "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollTransfer_WithError_Success() {
	s.env.OnActivity(activities.PluginPollTransferStatusActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.PollTransferStatusRequest) (*models.PollTransferStatusResponse, error) {
		s.Equal(s.connectorID, req.ConnectorID)
		s.Equal("test-transfer", req.Req.TransferID)
		return &models.PollTransferStatusResponse{
			Error: pointer.For("test-error"),
		}, nil
	})

	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, "test-schedule").Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteActivity, mock.Anything, "test-schedule").Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		s.NotNil(task.Error)
		s.ErrorContains(task.Error, "test-error")
		return nil
	})

	s.env.ExecuteWorkflow(RunPollTransfer, PollTransfer{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:       s.connectorID,
		PaymentInitiation: &s.paymentInitiationTransfer,
		TransferID:        "test-transfer",
		ScheduleID:        "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollTransfer_PluginPollTransferStatus_Error() {
	s.env.OnActivity(activities.PluginPollTransferStatusActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "PLUGIN", fmt.Errorf("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		s.NotNil(task.Error)
		s.ErrorContains(task.Error, "test")
		return nil
	})

	s.env.ExecuteWorkflow(RunPollTransfer, PollTransfer{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:       s.connectorID,
		PaymentInitiation: &s.paymentInitiationTransfer,
		TransferID:        "test-transfer",
		ScheduleID:        "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollTransfer_StoragePaymentsStore_Error() {
	s.env.OnActivity(activities.PluginPollTransferStatusActivity, mock.Anything, mock.Anything).Once().Return(
		&models.PollTransferStatusResponse{
			Payment: &s.pspPayment,
		},
		nil,
	)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", fmt.Errorf("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		s.ErrorContains(task.Error, "test")
		return nil
	})

	s.env.ExecuteWorkflow(RunPollTransfer, PollTransfer{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:       s.connectorID,
		PaymentInitiation: &s.paymentInitiationTransfer,
		TransferID:        "test-transfer",
		ScheduleID:        "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollTransfer_StoragePaymentInitiationsRelatedPaymentsStore_Error() {
	s.env.OnActivity(activities.PluginPollTransferStatusActivity, mock.Anything, mock.Anything).Once().Return(
		&models.PollTransferStatusResponse{
			Payment: &s.pspPayment,
		},
		nil,
	)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsRelatedPaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", fmt.Errorf("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		s.ErrorContains(task.Error, "test")
		return nil
	})

	s.env.ExecuteWorkflow(RunPollTransfer, PollTransfer{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:       s.connectorID,
		PaymentInitiation: &s.paymentInitiationTransfer,
		TransferID:        "test-transfer",
		ScheduleID:        "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollTransfer_StoragePaymentInitiationsAdjustmentsStore_Error() {
	s.env.OnActivity(activities.PluginPollTransferStatusActivity, mock.Anything, mock.Anything).Once().Return(
		&models.PollTransferStatusResponse{
			Payment: &s.pspPayment,
		},
		nil,
	)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsRelatedPaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", fmt.Errorf("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		s.ErrorContains(task.Error, "test")
		return nil
	})

	s.env.ExecuteWorkflow(RunPollTransfer, PollTransfer{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:       s.connectorID,
		PaymentInitiation: &s.paymentInitiationTransfer,
		TransferID:        "test-transfer",
		ScheduleID:        "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollTransfer_TemporalDeleteSchedule_Error() {
	s.env.OnActivity(activities.PluginPollTransferStatusActivity, mock.Anything, mock.Anything).Once().Return(
		&models.PollTransferStatusResponse{
			Payment: &s.pspPayment,
		},
		nil,
	)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsRelatedPaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, "test-schedule").Once().Return(
		temporal.NewNonRetryableApplicationError("test", "TEMPORAL", fmt.Errorf("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		s.ErrorContains(task.Error, "test")
		return nil
	})

	s.env.ExecuteWorkflow(RunPollTransfer, PollTransfer{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:       s.connectorID,
		PaymentInitiation: &s.paymentInitiationTransfer,
		TransferID:        "test-transfer",
		ScheduleID:        "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollTransfer_StorageSchedulesDelete_Error() {
	s.env.OnActivity(activities.PluginPollTransferStatusActivity, mock.Anything, mock.Anything).Once().Return(
		&models.PollTransferStatusResponse{
			Payment: &s.pspPayment,
		},
		nil,
	)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsRelatedPaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, "test-schedule").Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteActivity, mock.Anything, "test-schedule").Once().Return(
		temporal.NewNonRetryableApplicationError("test", "TEMPORAL", fmt.Errorf("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		s.ErrorContains(task.Error, "test")
		return nil
	})

	s.env.ExecuteWorkflow(RunPollTransfer, PollTransfer{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:       s.connectorID,
		PaymentInitiation: &s.paymentInitiationTransfer,
		TransferID:        "test-transfer",
		ScheduleID:        "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollTransfer_StorageTasksStore_Error() {
	s.env.OnActivity(activities.PluginPollTransferStatusActivity, mock.Anything, mock.Anything).Once().Return(
		&models.PollTransferStatusResponse{
			Payment: &s.pspPayment,
		},
		nil,
	)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsRelatedPaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, "test-schedule").Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteActivity, mock.Anything, "test-schedule").Once().Return(
		temporal.NewNonRetryableApplicationError("test", "TEMPORAL", fmt.Errorf("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", fmt.Errorf("test")),
	)

	s.env.ExecuteWorkflow(RunPollTransfer, PollTransfer{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:       s.connectorID,
		PaymentInitiation: &s.paymentInitiationTransfer,
		TransferID:        "test-transfer",
		ScheduleID:        "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}
