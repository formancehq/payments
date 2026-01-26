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

func (s *UnitTestSuite) newOrderID() models.OrderID {
	return models.OrderID{
		Reference:   "test-order-" + uuid.New().String()[:8],
		ConnectorID: s.connectorID,
	}
}

func (s *UnitTestSuite) newOrder(orderID models.OrderID, tif models.TimeInForce, expiresAt *time.Time) models.Order {
	now := s.env.Now().UTC()
	return models.Order{
		ID:                  orderID,
		ConnectorID:         s.connectorID,
		Reference:           orderID.Reference,
		CreatedAt:           now,
		UpdatedAt:           now,
		Direction:           models.ORDER_DIRECTION_BUY,
		SourceAsset:         "USD/2",
		TargetAsset:         "BTC/8",
		Type:                models.ORDER_TYPE_LIMIT,
		Status:              models.ORDER_STATUS_PENDING,
		BaseQuantityOrdered: big.NewInt(100000000), // 1 BTC
		BaseQuantityFilled:  big.NewInt(0),
		LimitPrice:          big.NewInt(5000000000000), // 50000 USD
		TimeInForce:         tif,
		ExpiresAt:           expiresAt,
		Metadata: map[string]string{
			"key": "value",
		},
	}
}

func (s *UnitTestSuite) newPSPOrder(orderID models.OrderID, status models.OrderStatus) models.PSPOrder {
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

// Test_CreateOrder_ImmediateFill_Success tests order creation with immediate fill
func (s *UnitTestSuite) Test_CreateOrder_ImmediateFill_Success() {
	orderID := s.newOrderID()
	order := s.newOrder(orderID, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED, nil)
	pspOrder := s.newPSPOrder(orderID, models.ORDER_STATUS_FILLED)

	s.env.OnActivity(activities.StorageOrdersGetActivity, mock.Anything, mock.Anything).Once().Return(&order, nil)
	s.env.OnActivity(activities.StorageOrdersUpdateStatusActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.OrdersUpdateStatusRequest) error {
		s.Equal(orderID, req.ID)
		s.Equal(models.ORDER_STATUS_PENDING, req.Status)
		return nil
	})
	s.env.OnActivity(activities.PluginCreateOrderActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CreateOrderRequest) (*models.CreateOrderResponse, error) {
		s.Equal(s.connectorID, req.ConnectorID)
		return &models.CreateOrderResponse{
			Order: &pspOrder,
		}, nil
	})
	s.env.OnActivity(activities.StorageOrdersUpsertActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, orders []models.Order) error {
		s.Equal(1, len(orders))
		s.Equal(orderID.Reference, orders[0].Reference)
		return nil
	})
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_SUCCEEDED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateOrder, CreateOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID: s.connectorID,
		OrderID:     orderID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

// Test_CreateOrder_WithPolling_Success tests order creation that requires polling
func (s *UnitTestSuite) Test_CreateOrder_WithPolling_Success() {
	orderID := s.newOrderID()
	order := s.newOrder(orderID, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED, nil)
	pollingOrderID := "exchange-order-123"

	s.env.OnActivity(activities.StorageOrdersGetActivity, mock.Anything, mock.Anything).Once().Return(&order, nil)
	s.env.OnActivity(activities.StorageOrdersUpdateStatusActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.OrdersUpdateStatusRequest) error {
		s.Equal(orderID, req.ID)
		s.Equal(models.ORDER_STATUS_PENDING, req.Status)
		return nil
	})
	s.env.OnActivity(activities.PluginCreateOrderActivity, mock.Anything, mock.Anything).Once().Return(&models.CreateOrderResponse{
		PollingOrderID: pointer.For(pollingOrderID),
	}, nil)
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, schedule models.Schedule) error {
		s.Contains(schedule.ID, "polling-order")
		s.Equal(s.connectorID, schedule.ConnectorID)
		return nil
	})
	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, options activities.ScheduleCreateOptions) error {
		s.Contains(options.ScheduleID, "polling-order")
		s.Equal(RunPollOrder, options.Action.Workflow)
		return nil
	})
	s.env.OnActivity(activities.StorageOrdersUpdateStatusActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.OrdersUpdateStatusRequest) error {
		s.Equal(orderID, req.ID)
		s.Equal(models.ORDER_STATUS_OPEN, req.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateOrder, CreateOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID: s.connectorID,
		OrderID:     orderID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

// Test_CreateOrder_FOK_Rejected tests Fill-Or-Kill order rejection (single attempt, no retry)
func (s *UnitTestSuite) Test_CreateOrder_FOK_Rejected() {
	orderID := s.newOrderID()
	order := s.newOrder(orderID, models.TIME_IN_FORCE_FILL_OR_KILL, nil)

	s.env.OnActivity(activities.StorageOrdersGetActivity, mock.Anything, mock.Anything).Once().Return(&order, nil)
	s.env.OnActivity(activities.StorageOrdersUpdateStatusActivity, mock.Anything, mock.Anything).Once().Return(nil) // PENDING
	s.env.OnActivity(activities.PluginCreateOrderActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("insufficient liquidity", "REJECTED", errors.New("insufficient liquidity")),
	)
	s.env.OnActivity(activities.StorageOrdersUpdateStatusActivity, mock.Anything, mock.Anything).Once().Return(nil) // FAILED
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateOrder, CreateOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID: s.connectorID,
		OrderID:     orderID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "order rejected")
}

// Test_CreateOrder_IOC_Rejected tests Immediate-Or-Cancel order rejection (single attempt, no retry)
func (s *UnitTestSuite) Test_CreateOrder_IOC_Rejected() {
	orderID := s.newOrderID()
	order := s.newOrder(orderID, models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL, nil)

	s.env.OnActivity(activities.StorageOrdersGetActivity, mock.Anything, mock.Anything).Once().Return(&order, nil)
	s.env.OnActivity(activities.StorageOrdersUpdateStatusActivity, mock.Anything, mock.Anything).Once().Return(nil) // PENDING
	s.env.OnActivity(activities.PluginCreateOrderActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("no immediate fill available", "REJECTED", errors.New("no immediate fill available")),
	)
	s.env.OnActivity(activities.StorageOrdersUpdateStatusActivity, mock.Anything, mock.Anything).Once().Return(nil) // FAILED
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateOrder, CreateOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID: s.connectorID,
		OrderID:     orderID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "order rejected")
}

// Test_CreateOrder_GTD_WithExpiration tests Good-Till-Date order with expiration time
func (s *UnitTestSuite) Test_CreateOrder_GTD_WithExpiration_Success() {
	orderID := s.newOrderID()
	expiresAt := s.env.Now().Add(24 * time.Hour)
	order := s.newOrder(orderID, models.TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME, &expiresAt)
	pspOrder := s.newPSPOrder(orderID, models.ORDER_STATUS_FILLED)

	s.env.OnActivity(activities.StorageOrdersGetActivity, mock.Anything, mock.Anything).Once().Return(&order, nil)
	s.env.OnActivity(activities.StorageOrdersUpdateStatusActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginCreateOrderActivity, mock.Anything, mock.Anything).Once().Return(&models.CreateOrderResponse{
		Order: &pspOrder,
	}, nil)
	s.env.OnActivity(activities.StorageOrdersUpsertActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_SUCCEEDED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateOrder, CreateOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID: s.connectorID,
		OrderID:     orderID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

// Test_CreateOrder_StorageOrdersGet_Error tests error handling when fetching order fails
func (s *UnitTestSuite) Test_CreateOrder_StorageOrdersGet_Error() {
	orderID := s.newOrderID()

	s.env.OnActivity(activities.StorageOrdersGetActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("order not found", "STORAGE", errors.New("order not found")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateOrder, CreateOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID: s.connectorID,
		OrderID:     orderID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "order not found")
}

// Test_CreateOrder_StorageOrdersUpdateStatus_Error tests error when updating order status fails
func (s *UnitTestSuite) Test_CreateOrder_StorageOrdersUpdateStatus_Error() {
	orderID := s.newOrderID()
	order := s.newOrder(orderID, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED, nil)

	s.env.OnActivity(activities.StorageOrdersGetActivity, mock.Anything, mock.Anything).Once().Return(&order, nil)
	s.env.OnActivity(activities.StorageOrdersUpdateStatusActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("storage error", "STORAGE", errors.New("storage error")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateOrder, CreateOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID: s.connectorID,
		OrderID:     orderID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "storage error")
}

// Test_CreateOrder_PluginCreateOrder_Error tests error handling when plugin call fails
func (s *UnitTestSuite) Test_CreateOrder_PluginCreateOrder_Error() {
	orderID := s.newOrderID()
	order := s.newOrder(orderID, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED, nil)

	s.env.OnActivity(activities.StorageOrdersGetActivity, mock.Anything, mock.Anything).Once().Return(&order, nil)
	s.env.OnActivity(activities.StorageOrdersUpdateStatusActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginCreateOrderActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("exchange unavailable", "PLUGIN", errors.New("exchange unavailable")),
	)
	s.env.OnActivity(activities.StorageOrdersUpdateStatusActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateOrder, CreateOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID: s.connectorID,
		OrderID:     orderID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "exchange unavailable")
}

// Test_CreateOrder_StorageSchedulesStore_Error tests error when storing schedule fails
func (s *UnitTestSuite) Test_CreateOrder_StorageSchedulesStore_Error() {
	orderID := s.newOrderID()
	order := s.newOrder(orderID, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED, nil)
	pollingOrderID := "exchange-order-123"

	s.env.OnActivity(activities.StorageOrdersGetActivity, mock.Anything, mock.Anything).Once().Return(&order, nil)
	s.env.OnActivity(activities.StorageOrdersUpdateStatusActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginCreateOrderActivity, mock.Anything, mock.Anything).Once().Return(&models.CreateOrderResponse{
		PollingOrderID: pointer.For(pollingOrderID),
	}, nil)
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("storage error", "STORAGE", errors.New("storage error")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateOrder, CreateOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID: s.connectorID,
		OrderID:     orderID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "storage error")
}

// Test_CreateOrder_TemporalScheduleCreate_Error tests error when creating Temporal schedule fails
func (s *UnitTestSuite) Test_CreateOrder_TemporalScheduleCreate_Error() {
	orderID := s.newOrderID()
	order := s.newOrder(orderID, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED, nil)
	pollingOrderID := "exchange-order-123"

	s.env.OnActivity(activities.StorageOrdersGetActivity, mock.Anything, mock.Anything).Once().Return(&order, nil)
	s.env.OnActivity(activities.StorageOrdersUpdateStatusActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginCreateOrderActivity, mock.Anything, mock.Anything).Once().Return(&models.CreateOrderResponse{
		PollingOrderID: pointer.For(pollingOrderID),
	}, nil)
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("schedule error", "SCHEDULE", errors.New("schedule error")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateOrder, CreateOrder{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID: s.connectorID,
		OrderID:     orderID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "schedule error")
}
