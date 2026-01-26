package workflow

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) newPollOrderID() models.OrderID {
	return models.OrderID{
		Reference:   "poll-test-order-" + uuid.New().String()[:8],
		ConnectorID: s.connectorID,
	}
}

func (s *UnitTestSuite) newPollOrderPSPOrder(orderID models.OrderID, status models.OrderStatus) models.PSPOrder {
	now := s.env.Now().UTC()
	return models.PSPOrder{
		Reference:           orderID.Reference,
		CreatedAt:           now,
		Direction:           models.ORDER_DIRECTION_BUY,
		SourceAsset:         "USD/2",
		TargetAsset:         "BTC/8",
		Type:                models.ORDER_TYPE_LIMIT,
		Status:              status,
		BaseQuantityOrdered: big.NewInt(100000000),
		BaseQuantityFilled:  big.NewInt(100000000),
		LimitPrice:          big.NewInt(5000000000000),
		TimeInForce:         models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
		Metadata: map[string]string{
			"key": "value",
		},
		Raw: []byte(`{}`),
	}
}

// Test_PollOrder_Filled_Success tests successful polling when order is filled
func (s *UnitTestSuite) Test_PollOrder_Filled_Success() {
	orderID := s.newPollOrderID()
	scheduleID := "polling-order-test-" + orderID.Reference
	pspOrder := s.newPollOrderPSPOrder(orderID, models.ORDER_STATUS_FILLED)

	s.env.OnActivity(activities.PluginPollOrderStatusActivity, mock.Anything, mock.Anything).Once().Return(&models.PollOrderStatusResponse{
		Order: &pspOrder,
	}, nil)
	s.env.OnActivity(activities.StorageOrdersUpsertActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, orders []models.Order) error {
		s.Equal(1, len(orders))
		s.Equal(orderID.Reference, orders[0].Reference)
		s.Equal(models.ORDER_STATUS_FILLED, orders[0].Status)
		return nil
	})
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, scheduleID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteActivity, mock.Anything, scheduleID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_SUCCEEDED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunPollOrder, PollOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		OrderID:        orderID,
		PollingOrderID: "exchange-order-123",
		ScheduleID:     scheduleID,
		TimeInForce:    models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
		ExpiresAt:      nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

// Test_PollOrder_NotReady_ContinuePolling tests when order is not yet final
func (s *UnitTestSuite) Test_PollOrder_NotReady_ContinuePolling() {
	orderID := s.newPollOrderID()
	scheduleID := "polling-order-test-" + orderID.Reference

	s.env.OnActivity(activities.PluginPollOrderStatusActivity, mock.Anything, mock.Anything).Once().Return(&models.PollOrderStatusResponse{
		Order: nil, // Not ready yet
		Error: nil,
	}, nil)

	s.env.ExecuteWorkflow(RunPollOrder, PollOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		OrderID:        orderID,
		PollingOrderID: "exchange-order-123",
		ScheduleID:     scheduleID,
		TimeInForce:    models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
		ExpiresAt:      nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

// Test_PollOrder_PartialFill_ContinuePolling tests when order is partially filled but not final
func (s *UnitTestSuite) Test_PollOrder_PartialFill_ContinuePolling() {
	orderID := s.newPollOrderID()
	scheduleID := "polling-order-test-" + orderID.Reference
	pspOrder := s.newPollOrderPSPOrder(orderID, models.ORDER_STATUS_PARTIALLY_FILLED)
	pspOrder.BaseQuantityFilled = big.NewInt(50000000) // Half filled

	s.env.OnActivity(activities.PluginPollOrderStatusActivity, mock.Anything, mock.Anything).Once().Return(&models.PollOrderStatusResponse{
		Order: &pspOrder,
	}, nil)
	s.env.OnActivity(activities.StorageOrdersUpsertActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunPollOrder, PollOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		OrderID:        orderID,
		PollingOrderID: "exchange-order-123",
		ScheduleID:     scheduleID,
		TimeInForce:    models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
		ExpiresAt:      nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

// Test_PollOrder_GTD_Expired tests GTD order expiration during polling
func (s *UnitTestSuite) Test_PollOrder_GTD_Expired() {
	orderID := s.newPollOrderID()
	scheduleID := "polling-order-test-" + orderID.Reference
	// Set expiration time in the past
	expiresAt := s.env.Now().Add(-1 * time.Hour)

	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, scheduleID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteActivity, mock.Anything, scheduleID).Once().Return(nil)
	s.env.OnActivity(activities.StorageOrdersUpdateStatusActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.OrdersUpdateStatusRequest) error {
		s.Equal(orderID, req.ID)
		s.Equal(models.ORDER_STATUS_EXPIRED, req.Status)
		return nil
	})
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_SUCCEEDED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunPollOrder, PollOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		OrderID:        orderID,
		PollingOrderID: "exchange-order-123",
		ScheduleID:     scheduleID,
		TimeInForce:    models.TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME,
		ExpiresAt:      &expiresAt,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

// Test_PollOrder_Cancelled_Success tests when order is cancelled on exchange
func (s *UnitTestSuite) Test_PollOrder_Cancelled_Success() {
	orderID := s.newPollOrderID()
	scheduleID := "polling-order-test-" + orderID.Reference
	pspOrder := s.newPollOrderPSPOrder(orderID, models.ORDER_STATUS_CANCELLED)

	s.env.OnActivity(activities.PluginPollOrderStatusActivity, mock.Anything, mock.Anything).Once().Return(&models.PollOrderStatusResponse{
		Order: &pspOrder,
	}, nil)
	s.env.OnActivity(activities.StorageOrdersUpsertActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, orders []models.Order) error {
		s.Equal(1, len(orders))
		s.Equal(models.ORDER_STATUS_CANCELLED, orders[0].Status)
		return nil
	})
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, scheduleID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteActivity, mock.Anything, scheduleID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_SUCCEEDED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunPollOrder, PollOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		OrderID:        orderID,
		PollingOrderID: "exchange-order-123",
		ScheduleID:     scheduleID,
		TimeInForce:    models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
		ExpiresAt:      nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

// Test_PollOrder_ExchangeError tests when exchange returns an error
func (s *UnitTestSuite) Test_PollOrder_ExchangeError() {
	orderID := s.newPollOrderID()
	scheduleID := "polling-order-test-" + orderID.Reference
	errorMsg := "Order rejected by exchange"

	s.env.OnActivity(activities.PluginPollOrderStatusActivity, mock.Anything, mock.Anything).Once().Return(&models.PollOrderStatusResponse{
		Order: nil,
		Error: pointer.For(errorMsg),
	}, nil)
	s.env.OnActivity(activities.StorageOrdersUpdateStatusActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.OrdersUpdateStatusRequest) error {
		s.Equal(orderID, req.ID)
		s.Equal(models.ORDER_STATUS_FAILED, req.Status)
		return nil
	})
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, scheduleID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteActivity, mock.Anything, scheduleID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunPollOrder, PollOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		OrderID:        orderID,
		PollingOrderID: "exchange-order-123",
		ScheduleID:     scheduleID,
		TimeInForce:    models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
		ExpiresAt:      nil,
	})

	// Workflow completes successfully - error is captured in task status
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

// Test_PollOrder_PluginPollOrderStatus_Error tests error handling when plugin poll fails
func (s *UnitTestSuite) Test_PollOrder_PluginPollOrderStatus_Error() {
	orderID := s.newPollOrderID()
	scheduleID := "polling-order-test-" + orderID.Reference

	s.env.OnActivity(activities.PluginPollOrderStatusActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("plugin error", "PLUGIN", errors.New("plugin error")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunPollOrder, PollOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		OrderID:        orderID,
		PollingOrderID: "exchange-order-123",
		ScheduleID:     scheduleID,
		TimeInForce:    models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
		ExpiresAt:      nil,
	})

	// Workflow completes successfully - error is captured in task status
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

// Test_PollOrder_TemporalScheduleDelete_Error tests error when deleting schedule fails
func (s *UnitTestSuite) Test_PollOrder_TemporalScheduleDelete_Error() {
	orderID := s.newPollOrderID()
	scheduleID := "polling-order-test-" + orderID.Reference
	pspOrder := s.newPollOrderPSPOrder(orderID, models.ORDER_STATUS_FILLED)

	s.env.OnActivity(activities.PluginPollOrderStatusActivity, mock.Anything, mock.Anything).Once().Return(&models.PollOrderStatusResponse{
		Order: &pspOrder,
	}, nil)
	s.env.OnActivity(activities.StorageOrdersUpsertActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, scheduleID).Once().Return(
		temporal.NewNonRetryableApplicationError("schedule error", "SCHEDULE", errors.New("schedule error")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunPollOrder, PollOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		OrderID:        orderID,
		PollingOrderID: "exchange-order-123",
		ScheduleID:     scheduleID,
		TimeInForce:    models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
		ExpiresAt:      nil,
	})

	// Workflow completes successfully - error is captured in task status
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

// Test_PollOrder_StorageSchedulesDelete_Error tests error when deleting storage schedule fails
func (s *UnitTestSuite) Test_PollOrder_StorageSchedulesDelete_Error() {
	orderID := s.newPollOrderID()
	scheduleID := "polling-order-test-" + orderID.Reference
	pspOrder := s.newPollOrderPSPOrder(orderID, models.ORDER_STATUS_FILLED)

	s.env.OnActivity(activities.PluginPollOrderStatusActivity, mock.Anything, mock.Anything).Once().Return(&models.PollOrderStatusResponse{
		Order: &pspOrder,
	}, nil)
	s.env.OnActivity(activities.StorageOrdersUpsertActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, scheduleID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteActivity, mock.Anything, scheduleID).Once().Return(
		temporal.NewNonRetryableApplicationError("storage error", "STORAGE", errors.New("storage error")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunPollOrder, PollOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		OrderID:        orderID,
		PollingOrderID: "exchange-order-123",
		ScheduleID:     scheduleID,
		TimeInForce:    models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
		ExpiresAt:      nil,
	})

	// Workflow completes successfully - error is captured in task status
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}
