package storage

import (
	"context"
	"encoding/json"
	"github.com/formancehq/go-libs/v3/platform/postgres"
	"github.com/pkg/errors"

	"github.com/formancehq/payments/internal/models"
	"github.com/uptrace/bun"
)

type state struct {
	bun.BaseModel `bun:"table:states"`

	ID          models.StateID     `bun:"id,pk,type:character varying,notnull"`
	ConnectorID models.ConnectorID `bun:"connector_id,type:character varying,notnull"`
	State       json.RawMessage    `bun:"state,type:json,notnull"`
}

func (s *store) StatesUpsert(ctx context.Context, state models.State) error {
	toInsert := fromStateModels(state)

	_, err := s.db.NewInsert().
		Model(&toInsert).
		On("CONFLICT (id) DO UPDATE").
		Set("state = EXCLUDED.state").
		Exec(ctx)
	return errors.Wrap(postgres.ResolveError(err), "failed to upsert state")
}

func (s *store) StatesGet(ctx context.Context, id models.StateID) (models.State, error) {
	var state state

	err := s.db.NewSelect().
		Model(&state).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return models.State{}, errors.Wrap(postgres.ResolveError(err), "failed to get state")
	}

	res := toStateModels(state)
	return res, nil
}

func (s *store) StatesDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*state)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)

	return errors.Wrap(postgres.ResolveError(err), "failed to delete state")
}

func fromStateModels(from models.State) state {
	return state{
		ID:          from.ID,
		ConnectorID: from.ConnectorID,
		State:       from.State,
	}
}

func toStateModels(from state) models.State {
	return models.State{
		ID:          from.ID,
		ConnectorID: from.ConnectorID,
		State:       from.State,
	}
}
