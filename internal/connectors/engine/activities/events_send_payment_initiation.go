package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) EventsSendPaymentInitiation(ctx context.Context, pi models.PaymentInitiation) error {
	return a.events.Publish(ctx, a.events.NewEventSavedPaymentInitiation(pi))
}

var EventsSendPaymentInitiationActivity = Activities{}.EventsSendPaymentInitiation

func EventsSendPaymentInitiation(ctx workflow.Context, pi models.PaymentInitiation) error {
	return executeActivity(ctx, EventsSendPaymentInitiationActivity, nil, pi)
}
