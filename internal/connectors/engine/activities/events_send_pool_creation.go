package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

// Deprecated
func (a Activities) EventsSendPoolCreation(ctx context.Context, pool models.Pool) error {
	return a.events.Publish(ctx, a.events.NewEventSavedPool(pool))
}

// Deprecated
var EventsSendPoolCreationActivity = Activities{}.EventsSendPoolCreation

// Deprecated
func EventsSendPoolCreation(ctx workflow.Context, pool models.Pool) error {
	return executeActivity(ctx, EventsSendPoolCreationActivity, nil, pool)
}
