package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) EventsSendConversion(ctx context.Context, conversion models.Conversion) error {
	return a.events.Publish(ctx, a.events.NewEventSavedConversion(conversion))
}

var EventsSendConversionActivity = Activities{}.EventsSendConversion

func EventsSendConversion(ctx workflow.Context, conversion models.Conversion) error {
	return executeActivity(ctx, EventsSendConversionActivity, nil, conversion)
}
