package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// TODO(polo): add tests for this file

type psuBankBridgeConnectionAttempt struct {
	bun.BaseModel `bun:"table:bank_bridge_connection_attempts"`

	// Mandatory fields
	ID          uuid.UUID                                   `bun:"id,pk,type:uuid,notnull"`
	PsuID       uuid.UUID                                   `bun:"psu_id,type:uuid,notnull"`
	ConnectorID models.ConnectorID                          `bun:"connector_id,type:character varying,notnull"`
	CreatedAt   time.Time                                   `bun:"created_at,type:timestamp without time zone,notnull"`
	Status      models.PSUBankBridgeConnectionAttemptStatus `bun:"status,type:text,notnull"`
	State       json.RawMessage                             `bun:"state,type:jsonb,nullzero"`

	// Optional fields
	ClientRedirectURL *string    `bun:"client_redirect_url,type:text,nullzero"`
	TemporaryToken    *string    `bun:"temporary_token,type:text,nullzero"`
	ExpiresAt         *time.Time `bun:"expires_at,type:timestamp without time zone,nullzero"`
	Error             *string    `bun:"error,type:text,nullzero"`
}

func (s *store) PSUBankBridgeConnectionAttemptsUpsert(ctx context.Context, from models.PSUBankBridgeConnectionAttempt) error {
	attempt, err := fromPsuBankBridgeConnectionAttemptsModels(from)
	if err != nil {
		return err
	}

	_, err = s.db.NewInsert().
		Model(&attempt).
		On("CONFLICT (id) DO UPDATE").
		Set("error = EXCLUDED.error").
		Set("status = EXCLUDED.status").
		Set("temporary_token = EXCLUDED.temporary_token").
		Set("expires_at = EXCLUDED.expires_at").
		Set("state = EXCLUDED.state").
		Exec(ctx)
	if err != nil {
		return e("upserting bank bridge connection attempt", err)
	}

	return nil
}

func (s *store) PSUBankBridgeConnectionAttemptsGet(ctx context.Context, id uuid.UUID) (*models.PSUBankBridgeConnectionAttempt, error) {
	attempt := psuBankBridgeConnectionAttempt{}
	err := s.db.NewSelect().
		Model(&attempt).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, e("getting bank bridge connection attempt", err)
	}

	return toPsuBankBridgeConnectionAttemptsModels(attempt)
}

type psuBankBridges struct {
	bun.BaseModel `bun:"table:psu_bank_bridges"`

	// Mandatory fields
	PsuID       uuid.UUID          `bun:"psu_id,pk,type:uuid,notnull"`
	ConnectorID models.ConnectorID `bun:"connector_id,pk,type:character varying,notnull"`

	// Optional fields
	AccessToken *string           `bun:"access_token,type:text,nullzero"`
	ExpiresAt   *time.Time        `bun:"expires_at,type:timestamp without time zone,nullzero"`
	Metadata    map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`
}

func (s *store) PSUBankBridgesUpsert(ctx context.Context, psuID uuid.UUID, from models.PSUBankBridge) error {
	bankBridge := fromPsuBankBridgesModels(from, psuID)

	_, err := s.db.NewInsert().
		Model(&bankBridge).
		On("CONFLICT (psu_id, connector_id) DO UPDATE").
		Set("access_token = EXCLUDED.access_token").
		Set("expires_at = EXCLUDED.expires_at").
		Set("metadata = EXCLUDED.metadata").
		Exec(ctx)
	if err != nil {
		return e("upserting bank bridge", err)
	}

	return nil
}

func (s *store) PSUBankBridgesGet(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) (*models.PSUBankBridge, error) {
	bankBridge := psuBankBridges{}
	err := s.db.NewSelect().
		Model(&bankBridge).
		Where("psu_id = ?", psuID).
		Where("connector_id = ?", connectorID).
		Scan(ctx)
	if err != nil {
		return nil, e("getting bank bridge", err)
	}

	connections := []psuBankBridgeConnections{}
	err = s.db.NewSelect().
		Model(&connections).
		Where("psu_id = ?", psuID).
		Where("connector_id = ?", connectorID).
		Scan(ctx)
	if err != nil {
		return nil, e("getting bank bridge", err)
	}

	return toPsuBankBridgesModels(bankBridge, connections), nil
}

type PSUBankBridgesQuery struct{}

type ListPSUBankBridgesQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PSUBankBridgesQuery]]

func NewListPSUBankBridgesQuery(opts bunpaginate.PaginatedQueryOptions[PSUBankBridgesQuery]) ListPSUBankBridgesQuery {
	return ListPSUBankBridgesQuery{
		Order:    bunpaginate.OrderAsc,
		PageSize: opts.PageSize,
		Options:  opts,
	}
}

func (s *store) psuBankBridgesQueryContext(qb query.Builder) (string, []any, error) {
	return qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch {
		case key == "connector_id", key == "psu_id":
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
			}
			return fmt.Sprintf("%s = ?", key), []any{value}, nil
		case metadataRegex.Match([]byte(key)):
			if operator != "$match" {
				return "", nil, fmt.Errorf("'metadata' column can only be used with $match: %w", ErrValidation)
			}
			match := metadataRegex.FindAllStringSubmatch(key, 3)

			key := "metadata"
			return key + " @> ?", []any{map[string]any{
				match[0][1]: value,
			}}, nil
		default:
			return "", nil, fmt.Errorf("unknown key '%s' when building query: %w", key, ErrValidation)
		}
	}))
}

func (s *store) PSUBankBridgesList(ctx context.Context, query ListPSUBankBridgesQuery) (*bunpaginate.Cursor[models.PSUBankBridge], error) {
	var (
		where string
		args  []any
		err   error
	)
	if query.Options.QueryBuilder != nil {
		where, args, err = s.psuBankBridgesQueryContext(query.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[PSUBankBridgesQuery], psuBankBridges](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PSUBankBridgesQuery]])(&query),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			if where != "" {
				query = query.Where(where, args...)
			}

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch psu bank bridges", err)
	}

	psuBankBridges := make([]psuBankBridges, len(cursor.Data))
	for i, p := range cursor.Data {
		psuBankBridges[i] = p
	}

	psuBankBridgesModels := make([]models.PSUBankBridge, len(psuBankBridges))
	for i, p := range psuBankBridges {
		bb := toPsuBankBridgesModels(p, []psuBankBridgeConnections{})
		psuBankBridgesModels[i] = *bb
	}

	return &bunpaginate.Cursor[models.PSUBankBridge]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     psuBankBridgesModels,
	}, nil
}

type psuBankBridgeConnections struct {
	bun.BaseModel `bun:"table:psu_bank_bridge_connections"`

	// Mandatory fields
	PsuID        uuid.UUID          `bun:"psu_id,pk,type:uuid,notnull"`
	ConnectorID  models.ConnectorID `bun:"connector_id,pk,type:character varying,notnull"`
	ConnectionID string             `bun:"connection_id,pk,type:character varying,notnull"`
	CreatedAt    time.Time          `bun:"created_at,type:timestamp without time zone,notnull"`

	// Optional fields
	AccessToken *string           `bun:"access_token,type:text,nullzero"`
	ExpiresAt   *time.Time        `bun:"expires_at,type:timestamp without time zone,nullzero"`
	Metadata    map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`
}

func (s *store) PSUBankBridgeConnectionsUpsert(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, from models.PSUBankBridgeConnection) error {
	connection := fromPsuBankBridgeConnectionsModels(from, psuID, connectorID)

	_, err := s.db.NewInsert().
		Model(&connection).
		On("CONFLICT (psu_id, connector_id, connection_id) DO UPDATE").
		Set("access_token = EXCLUDED.access_token").
		Set("expires_at = EXCLUDED.expires_at").
		Set("metadata = EXCLUDED.metadata").
		Exec(ctx)
	if err != nil {
		return e("upserting bank bridge connection", err)
	}

	return nil
}

func (s *store) PSUBankBridgeConnectionsGet(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string) (*models.PSUBankBridgeConnection, error) {
	connection := psuBankBridgeConnections{}
	err := s.db.NewSelect().
		Model(&connection).
		Where("psu_id = ?", psuID).
		Where("connector_id = ?", connectorID).
		Where("connection_id = ?", connectionID).
		Scan(ctx)
	if err != nil {
		return nil, e("getting bank bridge connection", err)
	}

	return toPsuBankBridgeConnectionsModels(connection), nil
}

func (s *store) PSUBankBridgeConnectionsGetFromConnectionID(ctx context.Context, connectorID models.ConnectorID, connectionID string) (*models.PSUBankBridgeConnection, uuid.UUID, error) {
	connection := psuBankBridgeConnections{}
	err := s.db.NewSelect().
		Model(&connection).
		Where("connector_id = ?", connectorID).
		Where("connection_id = ?", connectionID).
		Scan(ctx)
	if err != nil {
		return nil, uuid.Nil, e("getting bank bridge connection", err)
	}

	return toPsuBankBridgeConnectionsModels(connection), connection.PsuID, nil
}

func (s *store) PSUBankBridgeConnectionsGetAll(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) ([]*models.PSUBankBridgeConnection, error) {
	connections := []psuBankBridgeConnections{}
	err := s.db.NewSelect().
		Model(&connections).
		Where("psu_id = ?", psuID).
		Where("connector_id = ?", connectorID).
		Scan(ctx)
	if err != nil {
		return nil, e("getting bank bridge connection", err)
	}

	connectionsModels := make([]*models.PSUBankBridgeConnection, len(connections))
	for i, connection := range connections {
		connectionsModels[i] = toPsuBankBridgeConnectionsModels(connection)
	}

	return connectionsModels, nil
}

func fromPsuBankBridgeConnectionAttemptsModels(from models.PSUBankBridgeConnectionAttempt) (psuBankBridgeConnectionAttempt, error) {
	token, expiresAt := fromTokenModels(from.TemporaryToken)

	state, err := json.Marshal(from.State)
	if err != nil {
		return psuBankBridgeConnectionAttempt{}, err
	}

	return psuBankBridgeConnectionAttempt{
		ID:                from.ID,
		PsuID:             from.PsuID,
		ConnectorID:       from.ConnectorID,
		CreatedAt:         time.New(from.CreatedAt),
		Status:            from.Status,
		State:             state,
		ClientRedirectURL: from.ClientRedirectURL,
		TemporaryToken:    token,
		ExpiresAt:         expiresAt,
		Error:             from.Error,
	}, nil
}

func toPsuBankBridgeConnectionAttemptsModels(from psuBankBridgeConnectionAttempt) (*models.PSUBankBridgeConnectionAttempt, error) {
	state := models.CallbackState{}
	if err := json.Unmarshal(from.State, &state); err != nil {
		return nil, err
	}

	return &models.PSUBankBridgeConnectionAttempt{
		ID:                from.ID,
		PsuID:             from.PsuID,
		ConnectorID:       from.ConnectorID,
		CreatedAt:         from.CreatedAt.Time,
		Status:            from.Status,
		State:             state,
		ClientRedirectURL: from.ClientRedirectURL,
		TemporaryToken:    toTokenModels(from.TemporaryToken, from.ExpiresAt),
		Error:             from.Error,
	}, nil
}

func fromPsuBankBridgesModels(from models.PSUBankBridge, psuID uuid.UUID) psuBankBridges {
	accessToken, expiresAt := fromTokenModels(from.AccessToken)

	return psuBankBridges{
		PsuID:       psuID,
		ConnectorID: from.ConnectorID,
		AccessToken: accessToken,
		ExpiresAt:   expiresAt,
		Metadata:    from.Metadata,
	}
}

func toPsuBankBridgesModels(from psuBankBridges, connections []psuBankBridgeConnections) *models.PSUBankBridge {
	connectionsModels := make([]*models.PSUBankBridgeConnection, len(connections))
	for i, connection := range connections {
		connectionsModels[i] = toPsuBankBridgeConnectionsModels(connection)
	}

	return &models.PSUBankBridge{
		ConnectorID: from.ConnectorID,
		AccessToken: toTokenModels(from.AccessToken, from.ExpiresAt),
		Metadata:    from.Metadata,
		Connections: connectionsModels,
	}
}

func fromPsuBankBridgeConnectionsModels(from models.PSUBankBridgeConnection, psuID uuid.UUID, connectorID models.ConnectorID) psuBankBridgeConnections {
	accessToken, expiresAt := fromTokenModels(from.AccessToken)

	return psuBankBridgeConnections{
		PsuID:        psuID,
		ConnectorID:  connectorID,
		ConnectionID: from.ConnectionID,
		CreatedAt:    time.New(from.CreatedAt),
		AccessToken:  accessToken,
		ExpiresAt:    expiresAt,
		Metadata:     from.Metadata,
	}
}

func toPsuBankBridgeConnectionsModels(from psuBankBridgeConnections) *models.PSUBankBridgeConnection {
	return &models.PSUBankBridgeConnection{
		ConnectionID: from.ConnectionID,
		CreatedAt:    from.CreatedAt.Time,
		AccessToken:  toTokenModels(from.AccessToken, from.ExpiresAt),
		Metadata:     from.Metadata,
	}
}

func fromTokenModels(from *models.Token) (*string, *time.Time) {
	if from == nil {
		return nil, nil
	}

	return &from.Token, pointer.For(time.New(from.ExpiresAt))
}

func toTokenModels(from *string, expiresAt *time.Time) *models.Token {
	if from == nil {
		return nil
	}

	token := &models.Token{
		Token: *from,
	}

	if expiresAt != nil {
		token.ExpiresAt = expiresAt.Time
	}

	return token
}
