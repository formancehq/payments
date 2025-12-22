package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

// Deprecated
func (a Activities) EventsSendAccount(ctx context.Context, account models.Account) error {
	return a.events.Publish(ctx, a.events.NewEventSavedAccounts(account))
}

// Deprecated
var EventsSendAccountActivity = Activities{}.EventsSendAccount

// Deprecated
func EventsSendAccount(ctx workflow.Context, account models.Account) error {
	return executeActivity(ctx, EventsSendAccountActivity, nil, account) //nolint:staticcheck // ignore deprecated
}
