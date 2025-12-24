package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

// Deprecated
func (a Activities) EventsSendBalance(ctx context.Context, balance models.Balance) error {
	return a.events.Publish(ctx, a.events.NewEventSavedBalances(balance))
}

// EventsSendBalanceActivity
// Deprecated
var EventsSendBalanceActivity = Activities{}.EventsSendBalance

// EventsSendBalance is a Temporal activity that sends a balance event.
// Deprecated
func EventsSendBalance(ctx workflow.Context, balance models.Balance) error {
	return executeActivity(ctx, EventsSendBalanceActivity, nil, balance) //nolint:staticcheck // ignore deprecated
}
