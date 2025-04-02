package activities

import (
	"context"
	"time"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) EventsSendConnectorReset(ctx context.Context, connectorID models.ConnectorID, at time.Time) error {
	return a.events.Publish(ctx, a.events.NewEventResetConnector(connectorID, at))
}

var EventsSendConnectorResetActivity = Activities{}.EventsSendConnectorReset

func EventsSendConnectorReset(ctx workflow.Context, connectorID models.ConnectorID, at time.Time) error {
	return executeActivity(ctx, EventsSendConnectorResetActivity, nil, connectorID, at)
}
