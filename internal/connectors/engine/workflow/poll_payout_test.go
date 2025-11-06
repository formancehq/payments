package workflow

import (
	"context"
	"fmt"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_PollPayout_WithPayment_Success() {
	s.env.OnActivity(activities.PluginPollPayoutStatusActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.PollPayoutStatusRequest) (*models.PollPayoutStatusResponse, error) {
		s.Equal(s.connectorID, req.ConnectorID)
		s.Equal("test-payout", req.Req.PayoutID)
		return &models.PollPayoutStatusResponse{
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
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.NotNil(req.Payment)
		return nil
	})
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.NotNil(req.PaymentInitiationRelatedPayment)
		return nil
	})
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, adj models.PaymentInitiationAdjustment) error {
		s.Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED, adj.Status)
		s.Equal(s.paymentInitiationID, adj.ID.PaymentInitiationID)
		return nil
	})
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.NotNil(req.PaymentInitiationAdjustment)
		return nil
	})
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, "test-schedule").Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteActivity, mock.Anything, "test-schedule").Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_SUCCEEDED, task.Status)
		return nil
	})
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.SendEventsRequest) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunPollPayout, PollPayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
		PayoutID:            "test-payout",
		ScheduleID:          "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollPayout_WithoutPaymentAndError_Success() {
	s.env.OnActivity(activities.PluginPollPayoutStatusActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.PollPayoutStatusRequest) (*models.PollPayoutStatusResponse, error) {
		s.Equal(s.connectorID, req.ConnectorID)
		s.Equal("test-payout", req.Req.PayoutID)
		return &models.PollPayoutStatusResponse{}, nil
	})

	s.env.ExecuteWorkflow(RunPollPayout, PollPayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
		PayoutID:            "test-payout",
		ScheduleID:          "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollPayout_WithError_Success() {
	s.env.OnActivity(activities.PluginPollPayoutStatusActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.PollPayoutStatusRequest) (*models.PollPayoutStatusResponse, error) {
		s.Equal(s.connectorID, req.ConnectorID)
		s.Equal("test-payout", req.Req.PayoutID)
		return &models.PollPayoutStatusResponse{
			Error: pointer.For("error-test"),
		}, nil
	})

	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, "test-schedule").Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteActivity, mock.Anything, "test-schedule").Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		s.NotNil(task.Error)
		s.ErrorContains(task.Error, "error-test")
		return nil
	})
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunPollPayout, PollPayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
		PayoutID:            "test-payout",
		ScheduleID:          "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollPayout_PluginPollPayoutStatus_Error() {
	s.env.OnActivity(activities.PluginPollPayoutStatusActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "PLUGIN", fmt.Errorf("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		s.NotNil(task.Error)
		s.ErrorContains(task.Error, "test")
		return nil
	})
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunPollPayout, PollPayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
		PayoutID:            "test-payout",
		ScheduleID:          "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollPayout_StoragePaymentsStore_Error() {
	s.env.OnActivity(activities.PluginPollPayoutStatusActivity, mock.Anything, mock.Anything).Once().Return(
		&models.PollPayoutStatusResponse{
			Payment: &s.pspPayment,
		},
		nil,
	)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", fmt.Errorf("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		s.ErrorContains(task.Error, "test")
		return nil
	})
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunPollPayout, PollPayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
		PayoutID:            "test-payout",
		ScheduleID:          "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollPayout_RunSendEvents_Error() {
	s.env.OnActivity(activities.PluginPollPayoutStatusActivity, mock.Anything, mock.Anything).Once().Return(
		&models.PollPayoutStatusResponse{
			Payment: &s.pspPayment,
		},
		nil,
	)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsRelatedPaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, relatedPayment activities.RelatedPayment) error {
		s.Equal(s.paymentInitiationID, relatedPayment.PiID)
		s.Equal(s.paymentPayoutID, relatedPayment.PID)
		return nil
	})
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "WORKFLOW", fmt.Errorf("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		s.ErrorContains(task.Error, "test")
		return nil
	})
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunPollPayout, PollPayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
		PayoutID:            "test-payout",
		ScheduleID:          "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollPayout_StoragePaymentInitiationsRelatedPaymentsStore_Error() {
	s.env.OnActivity(activities.PluginPollPayoutStatusActivity, mock.Anything, mock.Anything).Once().Return(
		&models.PollPayoutStatusResponse{
			Payment: &s.pspPayment,
		},
		nil,
	)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsRelatedPaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", fmt.Errorf("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		s.ErrorContains(task.Error, "test")
		return nil
	})
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunPollPayout, PollPayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
		PayoutID:            "test-payout",
		ScheduleID:          "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollPayout_StoragePaymentInitiationsAdjustmentsStore_Error() {
	s.env.OnActivity(activities.PluginPollPayoutStatusActivity, mock.Anything, mock.Anything).Once().Return(
		&models.PollPayoutStatusResponse{
			Payment: &s.pspPayment,
		},
		nil,
	)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsRelatedPaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", fmt.Errorf("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		s.ErrorContains(task.Error, "test")
		return nil
	})
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunPollPayout, PollPayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
		PayoutID:            "test-payout",
		ScheduleID:          "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollPayout_TemporalDeleteSchedule_Error() {
	s.env.OnActivity(activities.PluginPollPayoutStatusActivity, mock.Anything, mock.Anything).Once().Return(
		&models.PollPayoutStatusResponse{
			Payment: &s.pspPayment,
		},
		nil,
	)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsRelatedPaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, "test-schedule").Once().Return(
		temporal.NewNonRetryableApplicationError("test", "TEMPORAL", fmt.Errorf("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		s.ErrorContains(task.Error, "test")
		return nil
	})
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunPollPayout, PollPayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
		PayoutID:            "test-payout",
		ScheduleID:          "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollPayout_StorageSchedulesDelete_Error() {
	s.env.OnActivity(activities.PluginPollPayoutStatusActivity, mock.Anything, mock.Anything).Once().Return(
		&models.PollPayoutStatusResponse{
			Payment: &s.pspPayment,
		},
		nil,
	)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsRelatedPaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, "test-schedule").Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteActivity, mock.Anything, "test-schedule").Once().Return(
		temporal.NewNonRetryableApplicationError("test", "TEMPORAL", fmt.Errorf("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		s.ErrorContains(task.Error, "test")
		return nil
	})
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunPollPayout, PollPayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
		PayoutID:            "test-payout",
		ScheduleID:          "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_PollPayout_StorageTasksStore_Error() {
	s.env.OnActivity(activities.PluginPollPayoutStatusActivity, mock.Anything, mock.Anything).Once().Return(
		&models.PollPayoutStatusResponse{
			Payment: &s.pspPayment,
		},
		nil,
	)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsRelatedPaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.SendEventsActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, "test-schedule").Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteActivity, mock.Anything, "test-schedule").Once().Return(
		temporal.NewNonRetryableApplicationError("test", "TEMPORAL", fmt.Errorf("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", fmt.Errorf("error-test")),
	)

	s.env.ExecuteWorkflow(RunPollPayout, PollPayout{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
		PayoutID:            "test-payout",
		ScheduleID:          "test-schedule",
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}
