package events

import (
	"time"

	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
)

type V3PoolMessagePayload struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"createdAt"`
	AccountIDs []string  `json:"accountIDs"`
}

type V2PoolMessagePayload struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"createdAt"`
	AccountIDs []string  `json:"accountIDs"`
}

func (e Events) NewEventSavedPool(pool models.Pool) []publish.EventMessage {
	return []publish.EventMessage{
		toV2PoolEvent(pool),
		toV3PoolEvent(pool),
	}
}

func toV3PoolEvent(pool models.Pool) publish.EventMessage {
	payload := V3PoolMessagePayload{
		ID:        pool.ID.String(),
		Name:      pool.Name,
		CreatedAt: pool.CreatedAt,
	}

	payload.AccountIDs = make([]string, len(pool.PoolAccounts))
	for i := range pool.PoolAccounts {
		payload.AccountIDs[i] = pool.PoolAccounts[i].String()
	}

	return publish.EventMessage{
		IdempotencyKey: pool.IdempotencyKey(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.V3EventVersion,
		Type:           events.V3EventTypeSavedPool,
		Payload:        payload,
	}
}

func toV2PoolEvent(pool models.Pool) publish.EventMessage {
	payload := V2PoolMessagePayload{
		ID:        pool.ID.String(),
		Name:      pool.Name,
		CreatedAt: pool.CreatedAt,
	}

	payload.AccountIDs = make([]string, len(pool.PoolAccounts))
	for i := range pool.PoolAccounts {
		payload.AccountIDs[i] = pool.PoolAccounts[i].String()
	}

	return publish.EventMessage{
		Date:    time.Now().UTC(),
		App:     events.EventApp,
		Version: events.V2EventVersion,
		Type:    events.V2EventTypeSavedPool,
		Payload: payload,
	}
}

type V3DeletePoolMessagePayload struct {
	CreatedAt time.Time `json:"createdAt"`
	ID        string    `json:"id"`
}

type V2DeletePoolMessagePayload struct {
	CreatedAt time.Time `json:"createdAt"`
	ID        string    `json:"id"`
}

func (e Events) NewEventDeletePool(id uuid.UUID, at time.Time) []publish.EventMessage {
	return []publish.EventMessage{
		toV2PoolDeleteEvent(id, at),
		toV3PoolDeleteEvent(id, at),
	}
}

func toV3PoolDeleteEvent(id uuid.UUID, at time.Time) publish.EventMessage {
	return publish.EventMessage{
		IdempotencyKey: id.String(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.V3EventVersion,
		Type:           events.V3EventTypeDeletePool,
		Payload: V3DeletePoolMessagePayload{
			CreatedAt: at,
			ID:        id.String(),
		},
	}
}

func toV2PoolDeleteEvent(id uuid.UUID, at time.Time) publish.EventMessage {
	return publish.EventMessage{
		Date:    time.Now().UTC(),
		App:     events.EventApp,
		Version: events.V2EventVersion,
		Type:    events.V2EventTypeDeletePool,
		Payload: V2DeletePoolMessagePayload{
			CreatedAt: at,
			ID:        id.String(),
		},
	}
}
