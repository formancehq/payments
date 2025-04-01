package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) EventsSendPaymentInitiationAdjustment(ctx context.Context, adj models.PaymentInitiationAdjustment, pi models.PaymentInitiation) error {
	return a.events.Publish(ctx, a.events.NewEventSavedPaymentInitiationAdjustment(adj, pi)...)
}

var EventsSendPaymentInitiationAdjustmentActivity = Activities{}.EventsSendPaymentInitiationAdjustment

func EventsSendPaymentInitiationAdjustment(ctx workflow.Context, adj models.PaymentInitiationAdjustment, pi models.PaymentInitiation) error {
	return executeActivity(ctx, EventsSendPaymentInitiationAdjustmentActivity, nil, adj, pi)
}
