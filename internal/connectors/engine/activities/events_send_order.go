package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type EventsSendOrderRequest struct {
	Order      models.Order
	Adjustment models.OrderAdjustment
}

func (a Activities) EventsSendOrder(ctx context.Context, req EventsSendOrderRequest) error {
	return a.events.Publish(ctx, a.events.NewEventSavedOrder(req.Order, req.Adjustment))
}

var EventsSendOrderActivity = Activities{}.EventsSendOrder

func EventsSendOrder(ctx workflow.Context, order models.Order, adjustment models.OrderAdjustment) error {
	return executeActivity(ctx, EventsSendOrderActivity, nil, EventsSendOrderRequest{
		Order:      order,
		Adjustment: adjustment,
	})
}
