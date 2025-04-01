package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) EventsSendPaymentInitiationRelatedPayment(
	ctx context.Context,
	relatedPayment models.PaymentInitiationRelatedPayments,
	pi models.PaymentInitiation,
	status models.PaymentInitiationAdjustmentStatus,
) error {
	return a.events.Publish(ctx, a.events.NewEventSavedPaymentInitiationRelatedPayment(relatedPayment, pi, status)...)
}

var EventsSendPaymentInitiationRelatedPaymentActivity = Activities{}.EventsSendPaymentInitiationRelatedPayment

func EventsSendPaymentInitiationRelatedPayment(
	ctx workflow.Context,
	relatedPayment models.PaymentInitiationRelatedPayments,
	pi models.PaymentInitiation,
	status models.PaymentInitiationAdjustmentStatus,
) error {
	return executeActivity(ctx, EventsSendPaymentInitiationRelatedPaymentActivity, nil, relatedPayment, pi, status)
}
