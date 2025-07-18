package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) EventsSendUserPendingDisconnect(ctx context.Context, userPendingDisconnect models.UserConnectionPendingDisconnect) error {
	return a.events.Publish(ctx, a.events.NewEventBankBridgeUserPendingDisconnect(userPendingDisconnect))
}

var EventsSendUserPendingDisconnectActivity = Activities{}.EventsSendUserPendingDisconnect

func EventsSendUserPendingDisconnect(ctx workflow.Context, userPendingDisconnect models.UserConnectionPendingDisconnect) error {
	return executeActivity(ctx, EventsSendUserPendingDisconnectActivity, nil, userPendingDisconnect)
}
