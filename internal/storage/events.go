package storage

import (
	"context"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/uptrace/bun"
)

type eventSent struct {
	bun.BaseModel `bun:"table:events_sent"`

	ID          models.EventID      `bun:"id,pk,type:character varying,notnull"`
	ConnectorID *models.ConnectorID `bun:"connector_id,type:character varying"`
	SentAt      time.Time           `bun:"sent_at,type:timestamp without time zone,notnull"`
}

func (s *store) EventsSentUpsert(ctx context.Context, event models.EventSent) error {
	toInsert := fromEventSentModel(event)

	_, err := s.db.NewInsert().
		Model(&toInsert).
		On("CONFLICT (id) DO NOTHING").
		Exec(ctx)

	return e("failed to insert event sent", err)
}

func (s *store) EventsSentGet(ctx context.Context, id models.EventID) (*models.EventSent, error) {
	var event eventSent

	err := s.db.NewSelect().
		Model(&event).
		Where("id = ?", id).
		Limit(1).
		Scan(ctx)

	if err != nil {
		return nil, e("failed to get event sent", err)
	}

	return pointer.For(toEventSentModel(event)), nil
}

func (s *store) EventsSentExists(ctx context.Context, id models.EventID) (bool, error) {
	exists, err := s.db.NewSelect().
		Model((*eventSent)(nil)).
		Where("id = ?", id).
		Limit(1).
		Exists(ctx)
	if err != nil {
		return false, e("failed to get event sent", err)
	}

	return exists, nil
}

func (s *store) EventsSentDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*eventSent)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)

	return e("failed to delete event sent", err)
}

// EventsSentDeleteFromConnectorIDBatch deletes a batch of events_sent for a given connector ID
// and returns the number of rows affected
func (s *store) EventsSentDeleteFromConnectorIDBatch(ctx context.Context, connectorID models.ConnectorID, batchSize int) (int, error) {
	result, err := s.db.NewDelete().
		Model((*eventSent)(nil)).
		Where("id IN (SELECT id FROM events_sent WHERE connector_id = ? LIMIT ?)", connectorID, batchSize).
		Exec(ctx)
	if err != nil {
		return 0, e("failed to delete events_sent batch", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, e("failed to get rows affected", err)
	}

	return int(rowsAffected), nil
}

func fromEventSentModel(from models.EventSent) eventSent {
	return eventSent{
		ID:          from.ID,
		ConnectorID: from.ConnectorID,
		SentAt:      time.New(from.SentAt),
	}
}

func toEventSentModel(from eventSent) models.EventSent {
	return models.EventSent{
		ID:          from.ID,
		ConnectorID: from.ConnectorID,
		SentAt:      from.SentAt.Time,
	}
}
