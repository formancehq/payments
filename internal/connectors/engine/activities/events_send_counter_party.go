package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) EventsSendCounterParty(ctx context.Context, counterParty models.CounterParty) error {
	return a.events.Publish(ctx, a.events.NewEventSavedCounterParty(counterParty))
}

var EventsSendCounterPartyActivity = Activities{}.EventsSendCounterParty

func EventsSendCounterParty(ctx workflow.Context, counterParty models.CounterParty) error {
	return executeActivity(ctx, EventsSendCounterPartyActivity, nil, counterParty)
}
