package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

// Deprecated
func (a Activities) EventsSendPaymentInitiationRelatedPayment(ctx context.Context, relatedPayment models.PaymentInitiationRelatedPayments) error {
	return a.events.Publish(ctx, a.events.NewEventSavedPaymentInitiationRelatedPayment(relatedPayment))
}

// Deprecated
var EventsSendPaymentInitiationRelatedPaymentActivity = Activities{}.EventsSendPaymentInitiationRelatedPayment

// Deprecated
func EventsSendPaymentInitiationRelatedPayment(ctx workflow.Context, relatedPayment models.PaymentInitiationRelatedPayments) error {
	return executeActivity(ctx, EventsSendPaymentInitiationRelatedPaymentActivity, nil, relatedPayment) //nolint:staticcheck // ignore deprecated
}
