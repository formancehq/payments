package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

// Deprecated
func (a Activities) EventsSendPaymentInitiationAdjustment(ctx context.Context, adj models.PaymentInitiationAdjustment) error {
	return a.events.Publish(ctx, a.events.NewEventSavedPaymentInitiationAdjustment(adj))
}

// Deprecated
var EventsSendPaymentInitiationAdjustmentActivity = Activities{}.EventsSendPaymentInitiationAdjustment

// Deprecated
func EventsSendPaymentInitiationAdjustment(ctx workflow.Context, adj models.PaymentInitiationAdjustment) error {
	return executeActivity(ctx, EventsSendPaymentInitiationAdjustmentActivity, nil, adj) //nolint:staticcheck // ignore deprecated
}
