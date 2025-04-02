package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) EventsSendPaymentInitiationAdjustment(ctx context.Context, adj models.PaymentInitiationAdjustment) error {
	return a.events.Publish(ctx, a.events.NewEventSavedPaymentInitiationAdjustment(adj))
}

var EventsSendPaymentInitiationAdjustmentActivity = Activities{}.EventsSendPaymentInitiationAdjustment

func EventsSendPaymentInitiationAdjustment(ctx workflow.Context, adj models.PaymentInitiationAdjustment) error {
	return executeActivity(ctx, EventsSendPaymentInitiationAdjustmentActivity, nil, adj)
}
