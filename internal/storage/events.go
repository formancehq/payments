package storage

import (
	"context"
	"github.com/formancehq/go-libs/v3/platform/postgres"
	"github.com/pkg/errors"

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

	return errors.Wrap(postgres.ResolveError(err), "failed to insert event sent")
}

func (s *store) EventsSentGet(ctx context.Context, id models.EventID) (*models.EventSent, error) {
	var event eventSent

	err := s.db.NewSelect().
		Model(&event).
		Where("id = ?", id).
		Limit(1).
		Scan(ctx)

	if err != nil {
		return nil, errors.Wrap(postgres.ResolveError(err), "failed to get event sent")
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
		return false, errors.Wrap(postgres.ResolveError(err), "failed to get event sent")
	}

	return exists, nil
}

func (s *store) EventsSentDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*eventSent)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)

	return errors.Wrap(postgres.ResolveError(err), "failed to delete event sent")
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
