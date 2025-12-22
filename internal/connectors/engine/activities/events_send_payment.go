package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type EventsSendPaymentRequest struct {
	Payment    models.Payment
	Adjustment models.PaymentAdjustment
}

// Deprecated
func (a Activities) EventsSendPayment(ctx context.Context, req EventsSendPaymentRequest) error {
	return a.events.Publish(ctx, a.events.NewEventSavedPayments(req.Payment, req.Adjustment))
}

// Deprecated
var EventsSendPaymentActivity = Activities{}.EventsSendPayment

// Deprecated
func EventsSendPayment(ctx workflow.Context, payment models.Payment, adjustment models.PaymentAdjustment) error {
	return executeActivity(ctx, EventsSendPaymentActivity, nil, EventsSendPaymentRequest{ //nolint:staticcheck // ignore deprecated
		Payment:    payment,
		Adjustment: adjustment,
	})
}
