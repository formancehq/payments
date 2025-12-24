package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

// Deprecated
func (a Activities) EventsSendPaymentInitiation(ctx context.Context, pi models.PaymentInitiation) error {
	return a.events.Publish(ctx, a.events.NewEventSavedPaymentInitiation(pi))
}

// Deprecated
var EventsSendPaymentInitiationActivity = Activities{}.EventsSendPaymentInitiation

// Deprecated
func EventsSendPaymentInitiation(ctx workflow.Context, pi models.PaymentInitiation) error {
	return executeActivity(ctx, EventsSendPaymentInitiationActivity, nil, pi) //nolint:staticcheck // ignore deprecated
}
