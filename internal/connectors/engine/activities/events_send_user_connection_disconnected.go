package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) EventsSendUserConnectionDisconnected(ctx context.Context, userConnectionDisconnected models.UserConnectionDisconnected) error {
	return a.events.Publish(ctx, a.events.NewEventOpenBankingUserConnectionDisconnected(userConnectionDisconnected))
}

var EventsSendUserConnectionDisconnectedActivity = Activities{}.EventsSendUserConnectionDisconnected

func EventsSendUserConnectionDisconnected(ctx workflow.Context, userConnectionDisconnected models.UserConnectionDisconnected) error {
	return executeActivity(ctx, EventsSendUserConnectionDisconnectedActivity, nil, userConnectionDisconnected)
}
