package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) EventsSendUserDisconnected(ctx context.Context, userDisconnected models.UserDisconnected) error {
	return a.events.Publish(ctx, a.events.NewEventOpenBankingUserDisconnected(userDisconnected))
}

var EventsSendUserDisconnectedActivity = Activities{}.EventsSendUserDisconnected

func EventsSendUserDisconnected(ctx workflow.Context, userDisconnected models.UserDisconnected) error {
	return executeActivity(ctx, EventsSendUserDisconnectedActivity, nil, userDisconnected)
}
