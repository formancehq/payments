package events

import (
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type connectorMessagePayload struct {
	CreatedAt   time.Time `json:"createdAt"`
	ConnectorID string    `json:"connectorID"`
}

func (e Events) NewEventResetConnector(connectorID models.ConnectorID, at time.Time) publish.EventMessage {
	return publish.EventMessage{
		IdempotencyKey: resetConnectorIdempotencyKey(connectorID, at),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeConnectorReset,
		Payload: connectorMessagePayload{
			CreatedAt:   at,
			ConnectorID: connectorID.String(),
		},
	}
}

func resetConnectorIdempotencyKey(connectorID models.ConnectorID, at time.Time) string {
	return fmt.Sprintf("%s-%s", connectorID.String(), at.Format(time.RFC3339Nano))
}
