package workflow

import (
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type CreateOrder struct {
	TaskID      models.TaskID
	ConnectorID models.ConnectorID
	OrderID     models.OrderID
}

func (w Workflow) runCreateOrder(
	ctx workflow.Context,
	createOrder CreateOrder,
) error {
	err := w.createOrder(ctx, createOrder)
	if err != nil {
		errUpdateTask := w.updateTasksError(
			ctx,
			createOrder.TaskID,
			&createOrder.ConnectorID,
			err,
		)
		if errUpdateTask != nil {
			return errUpdateTask
		}

		return err
	}

	return nil
}

func (w Workflow) createOrder(
	ctx workflow.Context,
	createOrder CreateOrder,
) error {
	// Get the order from storage
	order, err := activities.StorageOrdersGet(
		infiniteRetryContext(ctx),
		createOrder.OrderID,
	)
	if err != nil {
		return err
	}

	// Build PSP order from internal order
	pspOrder := models.ToPSPOrder(order)

	// Determine retry context based on TimeInForce
	// FOK and IOC orders must NOT retry - they are one-shot only
	activityCtx := w.getOrderActivityContext(ctx, order.TimeInForce, order.ExpiresAt)

	// Update order status to PENDING (being sent to exchange)
	err = activities.StorageOrdersUpdateStatus(
		infiniteRetryContext(ctx),
		createOrder.OrderID,
		models.ORDER_STATUS_PENDING,
	)
	if err != nil {
		return err
	}

	// Send order to exchange
	createOrderResponse, errPlugin := activities.PluginCreateOrder(
		activityCtx,
		createOrder.ConnectorID,
		models.CreateOrderRequest{
			Order: pspOrder,
		},
	)

	switch errPlugin {
	case nil:
		if createOrderResponse.Order != nil {
			// Order immediately returned (e.g., market order filled)
			updatedOrder, err := models.FromPSPOrderToOrder(*createOrderResponse.Order, createOrder.ConnectorID)
			if err != nil {
				return temporal.NewNonRetryableApplicationError(
					"failed to translate psp order",
					ErrValidation,
					err,
				)
			}

			if err := activities.StorageOrdersUpsert(
				infiniteRetryContext(ctx),
				[]models.Order{updatedOrder},
			); err != nil {
				return err
			}

			return w.updateTaskSuccess(
				ctx,
				createOrder.TaskID,
				&createOrder.ConnectorID,
				updatedOrder.ID.String(),
			)
		}

		if createOrderResponse.PollingOrderID != nil {
			// Order not yet filled, need to poll for status
			config, err := w.connectors.GetConfig(createOrder.ConnectorID)
			if err != nil {
				return err
			}

			scheduleID := fmt.Sprintf("polling-order-%s-%s-%s", w.stack, createOrder.ConnectorID.String(), *createOrderResponse.PollingOrderID)

			err = activities.StorageSchedulesStore(
				infiniteRetryContext(ctx),
				models.Schedule{
					ID:          scheduleID,
					ConnectorID: createOrder.ConnectorID,
					CreatedAt:   workflow.Now(ctx).UTC(),
				})
			if err != nil {
				return err
			}

			err = activities.TemporalScheduleCreate(
				infiniteRetryContext(ctx),
				activities.ScheduleCreateOptions{
					ScheduleID: scheduleID,
					Interval: client.ScheduleIntervalSpec{
						Every: config.PollingPeriod,
					},
					Action: client.ScheduleWorkflowAction{
						Workflow: RunPollOrder,
						Args: []interface{}{
							PollOrder{
								TaskID:         createOrder.TaskID,
								ConnectorID:    createOrder.ConnectorID,
								OrderID:        createOrder.OrderID,
								PollingOrderID: *createOrderResponse.PollingOrderID,
								ScheduleID:     scheduleID,
								TimeInForce:    order.TimeInForce,
								ExpiresAt:      order.ExpiresAt,
							},
						},
						TaskQueue: w.getDefaultTaskQueue(),
						TypedSearchAttributes: temporal.NewSearchAttributes(
							temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet(scheduleID),
							temporal.NewSearchAttributeKeyKeyword(SearchAttributeStack).ValueSet(w.stack),
						),
					},
					Overlap:            enums.SCHEDULE_OVERLAP_POLICY_SKIP,
					TriggerImmediately: true,
					SearchAttributes: map[string]any{
						SearchAttributeScheduleID: scheduleID,
						SearchAttributeStack:      w.stack,
					},
				},
			)
			if err != nil {
				return err
			}

			// Update order to OPEN since it's been submitted
			err = activities.StorageOrdersUpdateStatus(
				infiniteRetryContext(ctx),
				createOrder.OrderID,
				models.ORDER_STATUS_OPEN,
			)
			if err != nil {
				return err
			}
		}

		return nil

	default:
		// Order creation failed
		cause := errorsutils.Cause(errPlugin)

		// Update order status to FAILED
		err := activities.StorageOrdersUpdateStatus(
			infiniteRetryContext(ctx),
			createOrder.OrderID,
			models.ORDER_STATUS_FAILED,
		)
		if err != nil {
			return err
		}

		// For FOK/IOC orders, failure is expected behavior - don't return error to Temporal
		// The order status has been updated, task should be marked accordingly
		if order.TimeInForce == models.TIME_IN_FORCE_FILL_OR_KILL ||
			order.TimeInForce == models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL {
			// Mark task as failed but don't return error (no retry needed)
			return temporal.NewNonRetryableApplicationError(
				fmt.Sprintf("order rejected: %v", cause),
				ErrOrderRejected,
				cause,
			)
		}

		return errPlugin
	}
}

// getOrderActivityContext returns the appropriate retry context based on TimeInForce
func (w Workflow) getOrderActivityContext(ctx workflow.Context, tif models.TimeInForce, expiresAt *time.Time) workflow.Context {
	switch tif {
	case models.TIME_IN_FORCE_FILL_OR_KILL, models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL:
		// FOK and IOC: NO RETRY - single attempt only
		// Retrying these would create duplicate orders on the exchange!
		return noRetryContext(ctx)

	case models.TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME:
		// GTD: Retry until expiration time
		if expiresAt != nil {
			return gtdRetryContext(ctx, *expiresAt)
		}
		return infiniteRetryContext(ctx)

	case models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED:
		fallthrough
	default:
		// GTC and default: Infinite retry with backoff
		return infiniteRetryContext(ctx)
	}
}

const (
	RunCreateOrder  = "CreateOrder"
	ErrOrderRejected = "ORDER_REJECTED"
)
