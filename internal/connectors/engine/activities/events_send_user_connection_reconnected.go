package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) EventsSendUserConnectionReconnected(ctx context.Context, userConnectionReconnected models.UserConnectionReconnected) error {
	return a.events.Publish(ctx, a.events.NewEventOpenBankingUserConnectionReconnected(userConnectionReconnected))
}

var EventsSendUserConnectionReconnectedActivity = Activities{}.EventsSendUserConnectionReconnected

func EventsSendUserConnectionReconnected(ctx workflow.Context, userConnectionReconnected models.UserConnectionReconnected) error {
	return executeActivity(ctx, EventsSendUserConnectionReconnectedActivity, nil, userConnectionReconnected)
}
