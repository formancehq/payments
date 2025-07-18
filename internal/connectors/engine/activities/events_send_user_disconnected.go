package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) EventsSendUserDisconnected(ctx context.Context, userDisconnected models.UserConnectionDisconnected) error {
	return a.events.Publish(ctx, a.events.NewEventBankBridgeUserDisconnected(userDisconnected))
}

var EventsSendUserDisconnectedActivity = Activities{}.EventsSendUserDisconnected

func EventsSendUserDisconnected(ctx workflow.Context, userDisconnected models.UserConnectionDisconnected) error {
	return executeActivity(ctx, EventsSendUserDisconnectedActivity, nil, userDisconnected)
}
