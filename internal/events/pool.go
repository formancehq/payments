package events

import (
	"time"

	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
)

type PoolMessagePayload struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	CreatedAt  time.Time       `json:"createdAt"`
	Type       models.PoolType `json:"type"`
	Query      map[string]any  `json:"query"`
	AccountIDs []string        `json:"accountIDs"`
}

func (e Events) NewEventSavedPool(pool models.Pool) publish.EventMessage {
	payload := PoolMessagePayload{
		ID:        pool.ID.String(),
		Name:      pool.Name,
		CreatedAt: pool.CreatedAt,
		Type:      pool.Type,
		Query:     pool.Query,
	}

	payload.AccountIDs = make([]string, len(pool.PoolAccounts))
	for i := range pool.PoolAccounts {
		payload.AccountIDs[i] = pool.PoolAccounts[i].String()
	}

	return publish.EventMessage{
		IdempotencyKey: pool.IdempotencyKey(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedPool,
		Payload:        payload,
	}
}

type DeletePoolMessagePayload struct {
	CreatedAt time.Time `json:"createdAt"`
	ID        string    `json:"id"`
}

func (e Events) NewEventDeletePool(id uuid.UUID) publish.EventMessage {
	return publish.EventMessage{
		IdempotencyKey: id.String(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeDeletePool,
		Payload: DeletePoolMessagePayload{
			CreatedAt: time.Now().UTC(),
			ID:        id.String(),
		},
	}
}
