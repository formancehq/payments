package activities

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) EventsSendPoolDeletion(ctx context.Context, id uuid.UUID, at time.Time) error {
	return a.events.Publish(ctx, a.events.NewEventDeletePool(id, at)...)
}

var EventsSendPoolDeletionActivity = Activities{}.EventsSendPoolDeletion

func EventsSendPoolDeletion(ctx workflow.Context, id uuid.UUID, at time.Time) error {
	return executeActivity(ctx, EventsSendPoolDeletionActivity, nil, id, at)
}
