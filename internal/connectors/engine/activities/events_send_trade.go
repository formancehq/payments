package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type EventsSendTradeRequest struct {
	Trade models.Trade
}

func (a Activities) EventsSendTrade(ctx context.Context, req EventsSendTradeRequest) error {
	return a.events.Publish(ctx, a.events.NewEventSavedTrades(req.Trade))
}

var EventsSendTradeActivity = Activities{}.EventsSendTrade

func EventsSendTrade(ctx workflow.Context, trade models.Trade) error {
	return executeActivity(ctx, EventsSendTradeActivity, nil, EventsSendTradeRequest{
		Trade: trade,
	})
}

