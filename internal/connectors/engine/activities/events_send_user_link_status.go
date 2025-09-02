package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) EventsSendUserLinkStatus(ctx context.Context, userLinkStatus models.UserLinkSessionFinished) error {
	return a.events.Publish(ctx, a.events.NewEventOpenBankingUserLinkStatus(userLinkStatus))
}

var EventsSendUserLinkStatusActivity = Activities{}.EventsSendUserLinkStatus

func EventsSendUserLinkStatus(ctx workflow.Context, userLinkStatus models.UserLinkSessionFinished) error {
	return executeActivity(ctx, EventsSendUserLinkStatusActivity, nil, userLinkStatus)
}
