package activities

import (
	"context"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

// Deprecated
func (a Activities) EventsSendPoolDeletion(ctx context.Context, id uuid.UUID) error {
	return a.events.Publish(ctx, a.events.NewEventDeletePool(id))
}

// Deprecated
var EventsSendPoolDeletionActivity = Activities{}.EventsSendPoolDeletion

// Deprecated
func EventsSendPoolDeletion(ctx workflow.Context, id uuid.UUID) error {
	return executeActivity(ctx, EventsSendPoolDeletionActivity, nil, id) //nolint:staticcheck // ignore deprecated
}
