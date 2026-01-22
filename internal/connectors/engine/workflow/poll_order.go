package workflow

import (
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type PollOrder struct {
	TaskID         models.TaskID
	ConnectorID    models.ConnectorID
	OrderID        models.OrderID
	PollingOrderID string
	ScheduleID     string
	TimeInForce    models.TimeInForce
	ExpiresAt      *time.Time
}

func (w Workflow) runPollOrder(
	ctx workflow.Context,
	pollOrder PollOrder,
) error {
	orderIDStr, err := w.pollOrder(ctx, pollOrder)
	if err != nil {
		return w.updateTasksError(
			ctx,
			pollOrder.TaskID,
			&pollOrder.ConnectorID,
			err,
		)
	}

	if orderIDStr != "" {
		return w.updateTaskSuccess(
			ctx,
			pollOrder.TaskID,
			&pollOrder.ConnectorID,
			orderIDStr,
		)
	}

	return nil
}

func (w Workflow) pollOrder(
	ctx workflow.Context,
	pollOrder PollOrder,
) (string, error) {
	// Check if GTD order has expired
	if pollOrder.TimeInForce == models.TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME && pollOrder.ExpiresAt != nil {
		if workflow.Now(ctx).After(*pollOrder.ExpiresAt) {
			// Order expired, clean up and mark as expired
			if err := w.cleanupPollOrderSchedule(ctx, pollOrder); err != nil {
				return "", err
			}

			err := activities.StorageOrdersUpdateStatus(
				infiniteRetryContext(ctx),
				pollOrder.OrderID,
				models.ORDER_STATUS_EXPIRED,
			)
			if err != nil {
				return "", err
			}

			return pollOrder.OrderID.String(), nil
		}
	}

	pollOrderStatusResponse, err := activities.PluginPollOrderStatus(
		infiniteRetryContext(ctx),
		pollOrder.ConnectorID,
		models.PollOrderStatusRequest{
			OrderID: pollOrder.PollingOrderID,
		},
	)
	if err != nil {
		return "", err
	}

	orderIDStr := ""
	var orderErr error

	switch {
	case pollOrderStatusResponse.Order == nil && pollOrderStatusResponse.Error == nil:
		// Order not yet in final state, waiting for next poll
		return "", nil

	case pollOrderStatusResponse.Order != nil:
		order, err := models.FromPSPOrderToOrder(*pollOrderStatusResponse.Order, pollOrder.ConnectorID)
		if err != nil {
			return "", temporal.NewNonRetryableApplicationError(
				"failed to translate psp order",
				ErrValidation,
				err,
			)
		}

		if err := activities.StorageOrdersUpsert(
			infiniteRetryContext(ctx),
			[]models.Order{order},
		); err != nil {
			return "", err
		}

		// Only clean up schedule if order is in final state
		if order.Status.IsFinal() {
			orderIDStr = order.ID.String()
		} else {
			// Not final, keep polling
			return "", nil
		}

	case pollOrderStatusResponse.Error != nil:
		// Order failed on exchange
		orderErr = fmt.Errorf("%s", *pollOrderStatusResponse.Error)

		// Update order status to FAILED
		err := activities.StorageOrdersUpdateStatus(
			infiniteRetryContext(ctx),
			pollOrder.OrderID,
			models.ORDER_STATUS_FAILED,
		)
		if err != nil {
			return "", err
		}

		orderIDStr = pollOrder.OrderID.String()
	}

	// Clean up schedule (order is done or failed)
	if err := w.cleanupPollOrderSchedule(ctx, pollOrder); err != nil {
		return "", err
	}

	return orderIDStr, orderErr
}

func (w Workflow) cleanupPollOrderSchedule(ctx workflow.Context, pollOrder PollOrder) error {
	if err := activities.TemporalScheduleDelete(
		infiniteRetryContext(ctx),
		pollOrder.ScheduleID,
	); err != nil {
		return err
	}

	if err := activities.StorageSchedulesDelete(
		infiniteRetryContext(ctx),
		pollOrder.ScheduleID,
	); err != nil {
		return err
	}

	return nil
}

const RunPollOrder = "PollOrder"
