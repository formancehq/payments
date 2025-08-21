package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type EventsSendPaymentDeletedRequest struct {
	PaymentID models.PaymentID
}

func (a Activities) EventsSendPaymentDeleted(ctx context.Context, req EventsSendPaymentDeletedRequest) error {
	return a.events.Publish(ctx, a.events.NewEventPaymentDeleted(req.PaymentID))
}

var EventsSendPaymentDeletedActivity = Activities{}.EventsSendPaymentDeleted

func EventsSendPaymentDeleted(ctx workflow.Context, paymentID models.PaymentID) error {
	return executeActivity(ctx, EventsSendPaymentDeletedActivity, nil, EventsSendPaymentDeletedRequest{
		PaymentID: paymentID,
	})
}
