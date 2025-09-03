package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) EventsSendUserConnectionDataSynced(ctx context.Context, userConnectionDataSynced models.UserConnectionDataSynced) error {
	return a.events.Publish(ctx, a.events.NewEventOpenBankingUserConnectionDataSynced(userConnectionDataSynced))
}

var EventsSendUserConnectionDataSyncedActivity = Activities{}.EventsSendUserConnectionDataSynced

func EventsSendUserConnectionDataSynced(ctx workflow.Context, userConnectionDataSynced models.UserConnectionDataSynced) error {
	return executeActivity(ctx, EventsSendUserConnectionDataSyncedActivity, nil, userConnectionDataSynced)
}
