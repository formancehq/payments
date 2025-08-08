package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	libtime "time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

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

func (s *store) PSUBankBridgeConnectionAttemptsUpdateStatus(ctx context.Context, id uuid.UUID, status models.PSUBankBridgeConnectionAttemptStatus, errMsg *string) error {
	_, err := s.db.NewUpdate().
		Model((*psuBankBridgeConnectionAttempt)(nil)).
		Set("status = ?", status).
		Set("error = ?", errMsg).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return e("updating bank bridge connection attempt status", err)
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

type PSUBankBridgeConnectionAttemptsQuery struct{}

type ListPSUBankBridgeConnectionAttemptsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PSUBankBridgeConnectionAttemptsQuery]]

func NewListPSUBankBridgeConnectionAttemptsQuery(opts bunpaginate.PaginatedQueryOptions[PSUBankBridgeConnectionAttemptsQuery]) ListPSUBankBridgeConnectionAttemptsQuery {
	return ListPSUBankBridgeConnectionAttemptsQuery{
		PageSize: opts.PageSize,
		Order:    bunpaginate.OrderAsc,
		Options:  opts,
	}
}

func (s *store) psuBankBridgeConnectionAttemptsQueryContext(qb query.Builder) (string, []any, error) {
	return qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch {
		case key == "id":
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
			}
			return fmt.Sprintf("%s = ?", key), []any{value}, nil
		case key == "status":
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
			}
			return fmt.Sprintf("%s = ?", key), []any{value}, nil
		default:
			return "", nil, fmt.Errorf("unknown key '%s' when building query: %w", key, ErrValidation)
		}
	}))
}

func (s *store) PSUBankBridgeConnectionAttemptsList(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, query ListPSUBankBridgeConnectionAttemptsQuery) (*bunpaginate.Cursor[models.PSUBankBridgeConnectionAttempt], error) {
	var (
		where string
		args  []any
		err   error
	)
	if query.Options.QueryBuilder != nil {
		where, args, err = s.psuBankBridgeConnectionAttemptsQueryContext(query.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[PSUBankBridgeConnectionAttemptsQuery], psuBankBridgeConnectionAttempt](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PSUBankBridgeConnectionAttemptsQuery]])(&query),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			if where != "" {
				query = query.Where(where, args...)
			}

			query = query.Where("psu_id = ?", psuID)
			query = query.Where("connector_id = ?", connectorID)

			query = query.Order("created_at DESC")

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch psu bank bridge connection attempts", err)
	}

	psuBankBridgeConnectionAttemptsModels := make([]models.PSUBankBridgeConnectionAttempt, len(cursor.Data))
	for i, attempt := range cursor.Data {
		res, err := toPsuBankBridgeConnectionAttemptsModels(attempt)
		if err != nil {
			return nil, e("failed to fetch psu bank bridge connection attempts", err)
		}
		psuBankBridgeConnectionAttemptsModels[i] = *res
	}

	return &bunpaginate.Cursor[models.PSUBankBridgeConnectionAttempt]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     psuBankBridgeConnectionAttemptsModels,
	}, nil
}

type psuBankBridges struct {
	bun.BaseModel `bun:"table:psu_bank_bridges"`

	// Mandatory fields
	PsuID       uuid.UUID          `bun:"psu_id,pk,type:uuid,notnull"`
	ConnectorID models.ConnectorID `bun:"connector_id,pk,type:character varying,notnull"`

	// Optional fields
	PSPUserID *string           `bun:"psp_user_id,type:text,nullzero"`
	Metadata  map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`

	// Scan only fields
	AccessToken *string    `bun:"access_token,type:text,nullzero,scanonly"`
	ExpiresAt   *time.Time `bun:"expires_at,type:timestamp without time zone,nullzero,scanonly"`
}

func (s *store) PSUBankBridgesUpsert(ctx context.Context, psuID uuid.UUID, from models.PSUBankBridge) error {
	bankBridge, token := fromPsuBankBridgesModels(from, psuID)

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	if token != nil {
		_, err = tx.NewInsert().
			Model(token).
			On("CONFLICT (psu_id, connector_id, connection_id) DO UPDATE").
			Set("access_token = EXCLUDED.access_token").
			Set("expires_at = EXCLUDED.expires_at").
			Exec(ctx)
		if err != nil {
			return e("upserting bank bridge access token", err)
		}
	}

	_, err = tx.NewInsert().
		Model(&bankBridge).
		On("CONFLICT (psu_id, connector_id) DO UPDATE").
		Set("psp_user_id = EXCLUDED.psp_user_id").
		Set("expires_at = EXCLUDED.expires_at").
		Set("metadata = EXCLUDED.metadata").
		Exec(ctx)
	if err != nil {
		return e("upserting bank bridge", err)
	}

	return e("failed to commit transactions", tx.Commit())
}

func (s *store) PSUBankBridgesGet(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) (*models.PSUBankBridge, error) {
	bankBridge := psuBankBridges{}
	err := s.db.NewSelect().
		Model(&bankBridge).
		Column("psu_bank_bridges.*", "psu_bank_bridge_access_tokens.access_token", "psu_bank_bridge_access_tokens.expires_at").
		Join("LEFT JOIN psu_bank_bridge_access_tokens ON psu_bank_bridges.psu_id = psu_bank_bridge_access_tokens.psu_id AND psu_bank_bridges.connector_id = psu_bank_bridge_access_tokens.connector_id").
		Where("psu_bank_bridges.psu_id = ?", psuID).
		Where("psu_bank_bridges.connector_id = ?", connectorID).
		Scan(ctx)
	if err != nil {
		return nil, e("getting bank bridge", err)
	}

	return toPsuBankBridgesModels(bankBridge), nil
}

// TODO(polo): tests
func (s *store) PSUBankBridgesGetByPSPUserID(ctx context.Context, pspUserID string, connectorID models.ConnectorID) (*models.PSUBankBridge, error) {
	bankBridge := psuBankBridges{}
	err := s.db.NewSelect().
		Model(&bankBridge).
		Column("psu_bank_bridges.*", "psu_bank_bridge_access_tokens.access_token", "psu_bank_bridge_access_tokens.expires_at").
		Join("LEFT JOIN psu_bank_bridge_access_tokens ON psu_bank_bridges.psu_id = psu_bank_bridge_access_tokens.psu_id AND psu_bank_bridges.connector_id = psu_bank_bridge_access_tokens.connector_id").
		Where("psu_bank_bridges.psp_user_id = ?", pspUserID).
		Where("psu_bank_bridges.connector_id = ?", connectorID).
		Scan(ctx)
	if err != nil {
		return nil, e("getting bank bridge", err)
	}

	return toPsuBankBridgesModels(bankBridge), nil
}

func (s *store) PSUBankBridgesDelete(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*psuBankBridges)(nil)).
		Where("psu_id = ?", psuID).
		Where("connector_id = ?", connectorID).
		Exec(ctx)
	if err != nil {
		return e("deleting bank bridge", err)
	}

	return nil
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
			return fmt.Sprintf("psu_bank_bridges.%s = ?", key), []any{value}, nil
		case metadataRegex.Match([]byte(key)):
			if operator != "$match" {
				return "", nil, fmt.Errorf("'metadata' column can only be used with $match: %w", ErrValidation)
			}
			match := metadataRegex.FindAllStringSubmatch(key, 3)

			key := "psu_bank_bridges.metadata"
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

			query = query.
				Column("psu_bank_bridges.*", "psu_bank_bridge_access_tokens.access_token", "psu_bank_bridge_access_tokens.expires_at").
				Join("LEFT JOIN psu_bank_bridge_access_tokens ON psu_bank_bridges.psu_id = psu_bank_bridge_access_tokens.psu_id AND psu_bank_bridges.connector_id = psu_bank_bridge_access_tokens.connector_id")

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch psu bank bridges", err)
	}

	psuBankBridgesModels := make([]models.PSUBankBridge, len(cursor.Data))
	for i, p := range cursor.Data {
		bb := toPsuBankBridgesModels(p)
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
	PsuID         uuid.UUID               `bun:"psu_id,pk,type:uuid,notnull"`
	ConnectorID   models.ConnectorID      `bun:"connector_id,pk,type:character varying,notnull"`
	ConnectionID  string                  `bun:"connection_id,pk,type:character varying,notnull"`
	CreatedAt     time.Time               `bun:"created_at,type:timestamp without time zone,notnull"`
	DataUpdatedAt time.Time               `bun:"data_updated_at,type:timestamp without time zone,notnull"`
	Status        models.ConnectionStatus `bun:"status,type:text,notnull"`

	// Optional fields
	Error    *string           `bun:"error,type:text,nullzero"`
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`

	// ScanOnly fields
	AccessToken *string    `bun:"access_token,type:text,nullzero,scanonly"`
	ExpiresAt   *time.Time `bun:"expires_at,type:timestamp without time zone,nullzero,scanonly"`
}

func (s *store) PSUBankBridgeConnectionsUpsert(ctx context.Context, psuID uuid.UUID, from models.PSUBankBridgeConnection) error {
	connection, token := fromPsuBankBridgeConnectionsModels(from, psuID)

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	if token != nil {
		_, err = tx.NewInsert().
			Model(token).
			On("CONFLICT (psu_id, connector_id, connection_id) DO UPDATE").
			Set("access_token = EXCLUDED.access_token").
			Set("expires_at = EXCLUDED.expires_at").
			Exec(ctx)
		if err != nil {
			return e("upserting bank bridge connection access token", err)
		}
	}

	_, err = tx.NewInsert().
		Model(&connection).
		On("CONFLICT (psu_id, connector_id, connection_id) DO UPDATE").
		Set("metadata = EXCLUDED.metadata").
		Set("status = EXCLUDED.status").
		Set("error = EXCLUDED.error").
		Exec(ctx)
	if err != nil {
		return e("upserting bank bridge connection", err)
	}

	return e("failed to commit transactions", tx.Commit())
}

func (s *store) PSUBankBridgeConnectionsUpdateLastDataUpdate(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string, updatedAt libtime.Time) error {
	_, err := s.db.NewUpdate().
		Model((*psuBankBridgeConnections)(nil)).
		Set("data_updated_at = ?", updatedAt).
		Where("psu_id = ?", psuID).
		Where("connector_id = ?", connectorID).
		Where("connection_id = ?", connectionID).
		Exec(ctx)
	if err != nil {
		return e("updating bank bridge connection last data update", err)
	}

	return nil
}

func (s *store) PSUBankBridgeConnectionsGet(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string) (*models.PSUBankBridgeConnection, error) {
	connection := psuBankBridgeConnections{}
	err := s.db.NewSelect().
		Model(&connection).
		Column("psu_bank_bridge_connections.*", "psu_bank_bridge_access_tokens.access_token", "psu_bank_bridge_access_tokens.expires_at").
		Join("LEFT JOIN psu_bank_bridge_access_tokens ON psu_bank_bridge_connections.psu_id = psu_bank_bridge_access_tokens.psu_id AND psu_bank_bridge_connections.connector_id = psu_bank_bridge_access_tokens.connector_id AND psu_bank_bridge_connections.connection_id = psu_bank_bridge_access_tokens.connection_id").
		Where("psu_bank_bridge_connections.psu_id = ?", psuID).
		Where("psu_bank_bridge_connections.connector_id = ?", connectorID).
		Where("psu_bank_bridge_connections.connection_id = ?", connectionID).
		Scan(ctx)
	if err != nil {
		return nil, e("getting bank bridge connection", err)
	}

	return pointer.For(toPsuBankBridgeConnectionsModels(connection)), nil
}

func (s *store) PSUBankBridgeConnectionsGetFromConnectionID(ctx context.Context, connectorID models.ConnectorID, connectionID string) (*models.PSUBankBridgeConnection, uuid.UUID, error) {
	connection := psuBankBridgeConnections{}
	err := s.db.NewSelect().
		Model(&connection).
		Column("psu_bank_bridge_connections.*", "psu_bank_bridge_access_tokens.access_token", "psu_bank_bridge_access_tokens.expires_at").
		Join("LEFT JOIN psu_bank_bridge_access_tokens ON psu_bank_bridge_connections.psu_id = psu_bank_bridge_access_tokens.psu_id AND psu_bank_bridge_connections.connector_id = psu_bank_bridge_access_tokens.connector_id AND psu_bank_bridge_connections.connection_id = psu_bank_bridge_access_tokens.connection_id").
		Where("psu_bank_bridge_connections.connector_id = ?", connectorID).
		Where("psu_bank_bridge_connections.connection_id = ?", connectionID).
		Scan(ctx)
	if err != nil {
		return nil, uuid.Nil, e("getting bank bridge connection", err)
	}

	return pointer.For(toPsuBankBridgeConnectionsModels(connection)), connection.PsuID, nil
}

func (s *store) PSUBankBridgeConnectionsDelete(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string) error {
	_, err := s.db.NewDelete().
		Model((*psuBankBridgeConnections)(nil)).
		Where("psu_id = ?", psuID).
		Where("connector_id = ?", connectorID).
		Where("connection_id = ?", connectionID).
		Exec(ctx)
	if err != nil {
		return e("deleting bank bridge connection", err)
	}

	return nil
}

type PsuBankBridgeConnectionsQuery struct{}

type ListPsuBankBridgeConnectionsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PsuBankBridgeConnectionsQuery]]

func NewListPsuBankBridgeConnectionsQuery(opts bunpaginate.PaginatedQueryOptions[PsuBankBridgeConnectionsQuery]) ListPsuBankBridgeConnectionsQuery {
	return ListPsuBankBridgeConnectionsQuery{
		PageSize: opts.PageSize,
		Order:    bunpaginate.OrderAsc,
		Options:  opts,
	}
}

func (s *store) psuBankBridgeConnectionsQueryContext(qb query.Builder) (string, []any, error) {
	return qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch {
		case key == "connection_id", key == "status":
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

func (s *store) PSUBankBridgeConnectionsList(ctx context.Context, psuID uuid.UUID, connectorID *models.ConnectorID, query ListPsuBankBridgeConnectionsQuery) (*bunpaginate.Cursor[models.PSUBankBridgeConnection], error) {
	var (
		where string
		args  []any
		err   error
	)
	if query.Options.QueryBuilder != nil {
		where, args, err = s.psuBankBridgeConnectionsQueryContext(query.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[PsuBankBridgeConnectionsQuery], psuBankBridgeConnections](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PsuBankBridgeConnectionsQuery]])(&query),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			if where != "" {
				query = query.Where(where, args...)
			}

			query = query.Where("psu_id = ?", psuID)

			if connectorID != nil {
				query = query.Where("connector_id = ?", connectorID)
			}

			query = query.Order("created_at DESC")

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch psu bank bridge connections", err)
	}

	psuBankBridgeConnections := make([]psuBankBridgeConnections, len(cursor.Data))
	copy(psuBankBridgeConnections, cursor.Data)

	psuBankBridgeConnectionsModels := make([]models.PSUBankBridgeConnection, len(psuBankBridgeConnections))
	for i, connection := range psuBankBridgeConnections {
		psuBankBridgeConnectionsModels[i] = toPsuBankBridgeConnectionsModels(connection)
	}

	return &bunpaginate.Cursor[models.PSUBankBridgeConnection]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     psuBankBridgeConnectionsModels,
	}, nil
}

type psuBankBridgeAccessTokens struct {
	bun.BaseModel `bun:"table:psu_bank_bridge_access_tokens"`

	// Mandatory fields
	PSUID        uuid.UUID          `bun:"psu_id,type:uuid,notnull"`
	ConnectorID  models.ConnectorID `bun:"connector_id,type:character varying,notnull"`
	ConnectionID *string            `bun:"connection_id,type:character varying,nullzero"`
	AccessToken  string             `bun:"access_token,type:text,notnull"`
	ExpiresAt    time.Time          `bun:"expires_at,type:timestamp without time zone,notnull"`
}

func fromPsuBankBridgeConnectionAttemptsModels(from models.PSUBankBridgeConnectionAttempt) (psuBankBridgeConnectionAttempt, error) {
	var token *string
	var expiresAt *time.Time
	if from.TemporaryToken != nil {
		t, e := fromTokenModels(*from.TemporaryToken)
		token = &t
		expiresAt = &e
	}

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

func fromPsuBankBridgesModels(from models.PSUBankBridge, psuID uuid.UUID) (psuBankBridges, *psuBankBridgeAccessTokens) {
	var token *psuBankBridgeAccessTokens
	if from.AccessToken != nil {
		accessToken, expiresAt := fromTokenModels(*from.AccessToken)

		token = &psuBankBridgeAccessTokens{
			PSUID:       psuID,
			ConnectorID: from.ConnectorID,
			AccessToken: accessToken,
			ExpiresAt:   expiresAt,
		}
	}

	return psuBankBridges{
		PsuID:       psuID,
		PSPUserID:   from.PSPUserID,
		ConnectorID: from.ConnectorID,
		Metadata:    from.Metadata,
	}, token
}

func toPsuBankBridgesModels(from psuBankBridges) *models.PSUBankBridge {
	return &models.PSUBankBridge{
		PsuID:       from.PsuID,
		ConnectorID: from.ConnectorID,
		PSPUserID:   from.PSPUserID,
		AccessToken: toTokenModels(from.AccessToken, from.ExpiresAt),
		Metadata:    from.Metadata,
	}
}

func fromPsuBankBridgeConnectionsModels(from models.PSUBankBridgeConnection, psuID uuid.UUID) (psuBankBridgeConnections, *psuBankBridgeAccessTokens) {
	var token *psuBankBridgeAccessTokens
	if from.AccessToken != nil {
		accessToken, expiresAt := fromTokenModels(*from.AccessToken)

		token = &psuBankBridgeAccessTokens{
			PSUID:        psuID,
			ConnectorID:  from.ConnectorID,
			ConnectionID: &from.ConnectionID,
			AccessToken:  accessToken,
			ExpiresAt:    expiresAt,
		}
	}

	return psuBankBridgeConnections{
		PsuID:         psuID,
		ConnectorID:   from.ConnectorID,
		ConnectionID:  from.ConnectionID,
		CreatedAt:     time.New(from.CreatedAt),
		DataUpdatedAt: time.New(from.DataUpdatedAt),
		Status:        from.Status,
		Error:         from.Error,
		Metadata:      from.Metadata,
	}, token
}

func toPsuBankBridgeConnectionsModels(from psuBankBridgeConnections) models.PSUBankBridgeConnection {
	return models.PSUBankBridgeConnection{
		ConnectorID:   from.ConnectorID,
		ConnectionID:  from.ConnectionID,
		CreatedAt:     from.CreatedAt.Time,
		DataUpdatedAt: from.DataUpdatedAt.Time,
		Status:        from.Status,
		Error:         from.Error,
		AccessToken:   toTokenModels(from.AccessToken, from.ExpiresAt),
		Metadata:      from.Metadata,
	}
}

func fromTokenModels(from models.Token) (string, time.Time) {
	return from.Token, time.New(from.ExpiresAt)
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
