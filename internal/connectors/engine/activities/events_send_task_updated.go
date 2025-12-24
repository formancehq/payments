package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

// Deprecated
func (a Activities) EventsSendTaskUpdated(ctx context.Context, task models.Task) error {
	return a.events.Publish(ctx, a.events.NewEventUpdatedTask(task))
}

// Deprecated
var EventsSendTaskUpdatedActivity = Activities{}.EventsSendTaskUpdated

// Deprecated
func EventsSendTaskUpdated(ctx workflow.Context, task models.Task) error {
	return executeActivity(ctx, EventsSendTaskUpdatedActivity, nil, task) //nolint:staticcheck // ignore deprecated
}
