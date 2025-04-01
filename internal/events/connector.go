package events

import (
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type V3ConnectorMessagePayload struct {
	CreatedAt   time.Time `json:"createdAt"`
	ConnectorID string    `json:"connectorID"`
}

type V2ConnectorMessagePayload struct {
	CreatedAt   time.Time `json:"createdAt"`
	ConnectorID string    `json:"connectorId"`
}

func (e Events) NewEventResetConnector(connectorID models.ConnectorID, at time.Time) []publish.EventMessage {
	return []publish.EventMessage{
		toV2ConnectorEvent(connectorID, at),
		toV3ConnectorEvent(connectorID, at),
	}
}

func toV3ConnectorEvent(connectorID models.ConnectorID, at time.Time) publish.EventMessage {
	return publish.EventMessage{
		IdempotencyKey: resetConnectorIdempotencyKey(connectorID, at),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.V3EventTypeConnectorReset,
		Payload: V3ConnectorMessagePayload{
			CreatedAt:   at,
			ConnectorID: connectorID.String(),
		},
	}
}

func toV2ConnectorEvent(connectorID models.ConnectorID, at time.Time) publish.EventMessage {
	return publish.EventMessage{
		Date:    time.Now().UTC(),
		App:     events.EventApp,
		Version: events.EventVersion,
		Type:    events.V2EventTypeConnectorReset,
		Payload: V2ConnectorMessagePayload{
			CreatedAt:   at,
			ConnectorID: connectorID.String(),
		},
	}
}

func resetConnectorIdempotencyKey(connectorID models.ConnectorID, at time.Time) string {
	return fmt.Sprintf("%s-%s", connectorID.String(), at.Format(time.RFC3339Nano))
}
