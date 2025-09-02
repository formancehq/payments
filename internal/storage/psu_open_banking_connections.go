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

type psuOpenBankingConnectionAttempt struct {
	bun.BaseModel `bun:"table:open_banking_connection_attempts"`

	// Mandatory fields
	ID          uuid.UUID                                    `bun:"id,pk,type:uuid,notnull"`
	PsuID       uuid.UUID                                    `bun:"psu_id,type:uuid,notnull"`
	ConnectorID models.ConnectorID                           `bun:"connector_id,type:character varying,notnull"`
	CreatedAt   time.Time                                    `bun:"created_at,type:timestamp without time zone,notnull"`
	Status      models.PSUOpenBankingConnectionAttemptStatus `bun:"status,type:text,notnull"`
	State       json.RawMessage                              `bun:"state,type:jsonb,nullzero"`

	// Optional fields
	ClientRedirectURL *string    `bun:"client_redirect_url,type:text,nullzero"`
	TemporaryToken    *string    `bun:"temporary_token,type:text,nullzero"`
	ExpiresAt         *time.Time `bun:"expires_at,type:timestamp without time zone,nullzero"`
	Error             *string    `bun:"error,type:text,nullzero"`
}

func (s *store) PSUOpenBankingConnectionAttemptsUpsert(ctx context.Context, from models.PSUOpenBankingConnectionAttempt) error {
	attempt, err := fromPsuOpenBankingConnectionAttemptsModels(from)
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
		return e("upserting open banking connection attempt", err)
	}

	return nil
}

func (s *store) PSUOpenBankingConnectionAttemptsUpdateStatus(ctx context.Context, id uuid.UUID, status models.PSUOpenBankingConnectionAttemptStatus, errMsg *string) error {
	_, err := s.db.NewUpdate().
		Model((*psuOpenBankingConnectionAttempt)(nil)).
		Set("status = ?", status).
		Set("error = ?", errMsg).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return e("updating open banking connection attempt status", err)
	}

	return nil
}

func (s *store) PSUOpenBankingConnectionAttemptsGet(ctx context.Context, id uuid.UUID) (*models.PSUOpenBankingConnectionAttempt, error) {
	attempt := psuOpenBankingConnectionAttempt{}
	err := s.db.NewSelect().
		Model(&attempt).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, e("getting open banking connection attempt", err)
	}

	return toPsuOpenBankingConnectionAttemptsModels(attempt)
}

type PSUOpenBankingConnectionAttemptsQuery struct{}

type ListPSUOpenBankingConnectionAttemptsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PSUOpenBankingConnectionAttemptsQuery]]

func NewListPSUOpenBankingConnectionAttemptsQuery(opts bunpaginate.PaginatedQueryOptions[PSUOpenBankingConnectionAttemptsQuery]) ListPSUOpenBankingConnectionAttemptsQuery {
	return ListPSUOpenBankingConnectionAttemptsQuery{
		PageSize: opts.PageSize,
		Order:    bunpaginate.OrderAsc,
		Options:  opts,
	}
}

func (s *store) psuOpenBankingConnectionAttemptsQueryContext(qb query.Builder) (string, []any, error) {
	return qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch key {
		case "id":
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
			}
			return fmt.Sprintf("%s = ?", key), []any{value}, nil
		case "status":
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
			}
			return fmt.Sprintf("%s = ?", key), []any{value}, nil
		default:
			return "", nil, fmt.Errorf("unknown key '%s' when building query: %w", key, ErrValidation)
		}
	}))
}

func (s *store) PSUOpenBankingConnectionAttemptsList(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, query ListPSUOpenBankingConnectionAttemptsQuery) (*bunpaginate.Cursor[models.PSUOpenBankingConnectionAttempt], error) {
	var (
		where string
		args  []any
		err   error
	)
	if query.Options.QueryBuilder != nil {
		where, args, err = s.psuOpenBankingConnectionAttemptsQueryContext(query.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[PSUOpenBankingConnectionAttemptsQuery], psuOpenBankingConnectionAttempt](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PSUOpenBankingConnectionAttemptsQuery]])(&query),
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
		return nil, e("failed to fetch psu open banking connection attempts", err)
	}

	psuOpenBankingConnectionAttemptsModels := make([]models.PSUOpenBankingConnectionAttempt, len(cursor.Data))
	for i, attempt := range cursor.Data {
		res, err := toPsuOpenBankingConnectionAttemptsModels(attempt)
		if err != nil {
			return nil, e("failed to fetch psu open banking connection attempts", err)
		}
		psuOpenBankingConnectionAttemptsModels[i] = *res
	}

	return &bunpaginate.Cursor[models.PSUOpenBankingConnectionAttempt]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     psuOpenBankingConnectionAttemptsModels,
	}, nil
}

type openBankingProviderPSUs struct {
	bun.BaseModel `bun:"table:open_banking_provider_psus,alias:open_banking_provider_psus"`

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

func (s *store) OpenBankingProviderPSUUpsert(ctx context.Context, psuID uuid.UUID, from models.OpenBankingProviderPSU) error {
	openBanking, token := fromPsuOpenBankingModels(from, psuID)

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
			return e("upserting open banking provider psu access token", err)
		}
	}

	_, err = tx.NewInsert().
		Model(&openBanking).
		On("CONFLICT (psu_id, connector_id) DO UPDATE").
		Set("psp_user_id = EXCLUDED.psp_user_id").
		Set("expires_at = EXCLUDED.expires_at").
		Set("metadata = EXCLUDED.metadata").
		Exec(ctx)
	if err != nil {
		return e("upserting open banking provider psu", err)
	}

	return e("failed to commit transactions", tx.Commit())
}

func (s *store) OpenBankingProviderPSUGet(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) (*models.OpenBankingProviderPSU, error) {
	openBanking := openBankingProviderPSUs{}
	err := s.db.NewSelect().
		Model(&openBanking).
		Column("open_banking_provider_psus.*", "psu_open_banking_access_tokens.access_token", "psu_open_banking_access_tokens.expires_at").
		Join("LEFT JOIN psu_open_banking_access_tokens ON open_banking_provider_psus.psu_id = psu_open_banking_access_tokens.psu_id AND open_banking_provider_psus.connector_id = psu_open_banking_access_tokens.connector_id").
		Where("open_banking_provider_psus.psu_id = ?", psuID).
		Where("open_banking_provider_psus.connector_id = ?", connectorID).
		Scan(ctx)

	if err != nil {
		return nil, e("getting open banking provider psu", err)
	}

	return toPsuOpenBankingModels(openBanking), nil
}

func (s *store) OpenBankingProviderPSUGetByPSPUserID(ctx context.Context, pspUserID string, connectorID models.ConnectorID) (*models.OpenBankingProviderPSU, error) {
	openBankingPSU := openBankingProviderPSUs{}

	err := s.db.NewSelect().
		Model(&openBankingPSU).
		Column("open_banking_provider_psus.*", "psu_open_banking_access_tokens.access_token", "psu_open_banking_access_tokens.expires_at").
		Join("LEFT JOIN psu_open_banking_access_tokens ON open_banking_provider_psus.psu_id = psu_open_banking_access_tokens.psu_id AND open_banking_provider_psus.connector_id = psu_open_banking_access_tokens.connector_id").
		Where("open_banking_provider_psus.psp_user_id = ?", pspUserID).
		Where("open_banking_provider_psus.connector_id = ?", connectorID).
		Scan(ctx)

	if err != nil {
		return nil, e("getting open banking provider PSU", err)
	}

	return toPsuOpenBankingModels(openBankingPSU), nil
}

func (s *store) OpenBankingProviderPSUDelete(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*openBankingProviderPSUs)(nil)).
		Where("psu_id = ?", psuID).
		Where("connector_id = ?", connectorID).
		Exec(ctx)
	if err != nil {
		return e("deleting open banking provider psu", err)
	}

	return nil
}

type OpenBankingProviderPSUQuery struct{}

type ListOpenBankingProviderPSUQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[OpenBankingProviderPSUQuery]]

func NewListOpenBankingProviderPSUQuery(opts bunpaginate.PaginatedQueryOptions[OpenBankingProviderPSUQuery]) ListOpenBankingProviderPSUQuery {
	return ListOpenBankingProviderPSUQuery{
		Order:    bunpaginate.OrderAsc,
		PageSize: opts.PageSize,
		Options:  opts,
	}
}

func (s *store) psuOpenBankingQueryContext(qb query.Builder) (string, []any, error) {
	return qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch {
		case key == "connector_id", key == "psu_id":
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
			}
			return fmt.Sprintf("open_banking_provider_psus.%s = ?", key), []any{value}, nil
		case metadataRegex.Match([]byte(key)):
			if operator != "$match" {
				return "", nil, fmt.Errorf("'metadata' column can only be used with $match: %w", ErrValidation)
			}
			match := metadataRegex.FindAllStringSubmatch(key, 3)

			key := "open_banking_provider_psus.metadata"
			return key + " @> ?", []any{map[string]any{
				match[0][1]: value,
			}}, nil
		default:
			return "", nil, fmt.Errorf("unknown key '%s' when building query: %w", key, ErrValidation)
		}
	}))
}

func (s *store) OpenBankingProviderPSUList(ctx context.Context, query ListOpenBankingProviderPSUQuery) (*bunpaginate.Cursor[models.OpenBankingProviderPSU], error) {
	var (
		where string
		args  []any
		err   error
	)
	if query.Options.QueryBuilder != nil {
		where, args, err = s.psuOpenBankingQueryContext(query.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[OpenBankingProviderPSUQuery], openBankingProviderPSUs](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[OpenBankingProviderPSUQuery]])(&query),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			if where != "" {
				query = query.Where(where, args...)
			}

			query = query.
				Column("open_banking_provider_psus.*", "psu_open_banking_access_tokens.access_token", "psu_open_banking_access_tokens.expires_at").
				Join("LEFT JOIN psu_open_banking_access_tokens ON open_banking_provider_psus.psu_id = psu_open_banking_access_tokens.psu_id AND open_banking_provider_psus.connector_id = psu_open_banking_access_tokens.connector_id")

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch open banking provider psu", err)
	}

	psuOpenBankingModels := make([]models.OpenBankingProviderPSU, len(cursor.Data))
	for i, p := range cursor.Data {
		bb := toPsuOpenBankingModels(p)
		psuOpenBankingModels[i] = *bb
	}

	return &bunpaginate.Cursor[models.OpenBankingProviderPSU]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     psuOpenBankingModels,
	}, nil
}

type psuOpenBankingConnections struct {
	bun.BaseModel `bun:"table:psu_open_banking_connections"`

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

func (s *store) PSUOpenBankingConnectionsUpsert(ctx context.Context, psuID uuid.UUID, from models.PSUOpenBankingConnection) error {
	connection, token := fromPsuOpenBankingConnectionsModels(from, psuID)

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
			return e("upserting open banking connection access token", err)
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
		return e("upserting open banking connection", err)
	}

	return e("failed to commit transactions", tx.Commit())
}

func (s *store) PSUOpenBankingConnectionsUpdateLastDataUpdate(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string, updatedAt libtime.Time) error {
	_, err := s.db.NewUpdate().
		Model((*psuOpenBankingConnections)(nil)).
		Set("data_updated_at = ?", updatedAt).
		Where("psu_id = ?", psuID).
		Where("connector_id = ?", connectorID).
		Where("connection_id = ?", connectionID).
		Exec(ctx)
	if err != nil {
		return e("updating open banking connection last data update", err)
	}

	return nil
}

func (s *store) PSUOpenBankingConnectionsGet(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string) (*models.PSUOpenBankingConnection, error) {
	connection := psuOpenBankingConnections{}
	err := s.db.NewSelect().
		Model(&connection).
		Column("psu_open_banking_connections.*", "psu_open_banking_access_tokens.access_token", "psu_open_banking_access_tokens.expires_at").
		Join("LEFT JOIN psu_open_banking_access_tokens ON psu_open_banking_connections.psu_id = psu_open_banking_access_tokens.psu_id AND psu_open_banking_connections.connector_id = psu_open_banking_access_tokens.connector_id AND psu_open_banking_connections.connection_id = psu_open_banking_access_tokens.connection_id").
		Where("psu_open_banking_connections.psu_id = ?", psuID).
		Where("psu_open_banking_connections.connector_id = ?", connectorID).
		Where("psu_open_banking_connections.connection_id = ?", connectionID).
		Scan(ctx)
	if err != nil {
		return nil, e("getting open banking connection", err)
	}

	return pointer.For(toPsuOpenBankingConnectionsModels(connection)), nil
}

func (s *store) PSUOpenBankingConnectionsGetFromConnectionID(ctx context.Context, connectorID models.ConnectorID, connectionID string) (*models.PSUOpenBankingConnection, uuid.UUID, error) {
	connection := psuOpenBankingConnections{}
	err := s.db.NewSelect().
		Model(&connection).
		Column("psu_open_banking_connections.*", "psu_open_banking_access_tokens.access_token", "psu_open_banking_access_tokens.expires_at").
		Join("LEFT JOIN psu_open_banking_access_tokens ON psu_open_banking_connections.psu_id = psu_open_banking_access_tokens.psu_id AND psu_open_banking_connections.connector_id = psu_open_banking_access_tokens.connector_id AND psu_open_banking_connections.connection_id = psu_open_banking_access_tokens.connection_id").
		Where("psu_open_banking_connections.connector_id = ?", connectorID).
		Where("psu_open_banking_connections.connection_id = ?", connectionID).
		Scan(ctx)
	if err != nil {
		return nil, uuid.Nil, e("getting open banking connection", err)
	}

	return pointer.For(toPsuOpenBankingConnectionsModels(connection)), connection.PsuID, nil
}

func (s *store) PSUOpenBankingConnectionsDelete(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string) error {
	_, err := s.db.NewDelete().
		Model((*psuOpenBankingConnections)(nil)).
		Where("psu_id = ?", psuID).
		Where("connector_id = ?", connectorID).
		Where("connection_id = ?", connectionID).
		Exec(ctx)
	if err != nil {
		return e("deleting open banking connection", err)
	}

	return nil
}

type PsuOpenBankingConnectionsQuery struct{}

type ListPsuOpenBankingConnectionsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PsuOpenBankingConnectionsQuery]]

func NewListPsuOpenBankingConnectionsQuery(opts bunpaginate.PaginatedQueryOptions[PsuOpenBankingConnectionsQuery]) ListPsuOpenBankingConnectionsQuery {
	return ListPsuOpenBankingConnectionsQuery{
		PageSize: opts.PageSize,
		Order:    bunpaginate.OrderAsc,
		Options:  opts,
	}
}

func (s *store) psuOpenBankingConnectionsQueryContext(qb query.Builder) (string, []any, error) {
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

func (s *store) PSUOpenBankingConnectionsList(ctx context.Context, psuID uuid.UUID, connectorID *models.ConnectorID, query ListPsuOpenBankingConnectionsQuery) (*bunpaginate.Cursor[models.PSUOpenBankingConnection], error) {
	var (
		where string
		args  []any
		err   error
	)
	if query.Options.QueryBuilder != nil {
		where, args, err = s.psuOpenBankingConnectionsQueryContext(query.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[PsuOpenBankingConnectionsQuery], psuOpenBankingConnections](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PsuOpenBankingConnectionsQuery]])(&query),
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
		return nil, e("failed to fetch psu open banking connections", err)
	}

	psuOpenBankingConnections := make([]psuOpenBankingConnections, len(cursor.Data))
	copy(psuOpenBankingConnections, cursor.Data)

	psuOpenBankingConnectionsModels := make([]models.PSUOpenBankingConnection, len(psuOpenBankingConnections))
	for i, connection := range psuOpenBankingConnections {
		psuOpenBankingConnectionsModels[i] = toPsuOpenBankingConnectionsModels(connection)
	}

	return &bunpaginate.Cursor[models.PSUOpenBankingConnection]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     psuOpenBankingConnectionsModels,
	}, nil
}

type psuOpenBankingAccessTokens struct {
	bun.BaseModel `bun:"table:psu_open_banking_access_tokens"`

	// Mandatory fields
	PSUID        uuid.UUID          `bun:"psu_id,type:uuid,notnull"`
	ConnectorID  models.ConnectorID `bun:"connector_id,type:character varying,notnull"`
	ConnectionID *string            `bun:"connection_id,type:character varying,nullzero"`
	AccessToken  string             `bun:"access_token,type:text,notnull"`
	ExpiresAt    time.Time          `bun:"expires_at,type:timestamp without time zone,notnull"`
}

func fromPsuOpenBankingConnectionAttemptsModels(from models.PSUOpenBankingConnectionAttempt) (psuOpenBankingConnectionAttempt, error) {
	var token *string
	var expiresAt *time.Time
	if from.TemporaryToken != nil {
		t, e := fromTokenModels(*from.TemporaryToken)
		token = &t
		expiresAt = &e
	}

	state, err := json.Marshal(from.State)
	if err != nil {
		return psuOpenBankingConnectionAttempt{}, err
	}

	return psuOpenBankingConnectionAttempt{
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

func toPsuOpenBankingConnectionAttemptsModels(from psuOpenBankingConnectionAttempt) (*models.PSUOpenBankingConnectionAttempt, error) {
	state := models.CallbackState{}
	if err := json.Unmarshal(from.State, &state); err != nil {
		return nil, err
	}

	return &models.PSUOpenBankingConnectionAttempt{
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

func fromPsuOpenBankingModels(from models.OpenBankingProviderPSU, psuID uuid.UUID) (openBankingProviderPSUs, *psuOpenBankingAccessTokens) {
	var token *psuOpenBankingAccessTokens
	if from.AccessToken != nil {
		accessToken, expiresAt := fromTokenModels(*from.AccessToken)

		token = &psuOpenBankingAccessTokens{
			PSUID:       psuID,
			ConnectorID: from.ConnectorID,
			AccessToken: accessToken,
			ExpiresAt:   expiresAt,
		}
	}

	return openBankingProviderPSUs{
		PsuID:       psuID,
		PSPUserID:   from.PSPUserID,
		ConnectorID: from.ConnectorID,
		Metadata:    from.Metadata,
	}, token
}

func toPsuOpenBankingModels(from openBankingProviderPSUs) *models.OpenBankingProviderPSU {
	return &models.OpenBankingProviderPSU{
		PsuID:       from.PsuID,
		ConnectorID: from.ConnectorID,
		PSPUserID:   from.PSPUserID,
		AccessToken: toTokenModels(from.AccessToken, from.ExpiresAt),
		Metadata:    from.Metadata,
	}
}

func fromPsuOpenBankingConnectionsModels(from models.PSUOpenBankingConnection, psuID uuid.UUID) (psuOpenBankingConnections, *psuOpenBankingAccessTokens) {
	var token *psuOpenBankingAccessTokens
	if from.AccessToken != nil {
		accessToken, expiresAt := fromTokenModels(*from.AccessToken)

		token = &psuOpenBankingAccessTokens{
			PSUID:        psuID,
			ConnectorID:  from.ConnectorID,
			ConnectionID: &from.ConnectionID,
			AccessToken:  accessToken,
			ExpiresAt:    expiresAt,
		}
	}

	return psuOpenBankingConnections{
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

func toPsuOpenBankingConnectionsModels(from psuOpenBankingConnections) models.PSUOpenBankingConnection {
	return models.PSUOpenBankingConnection{
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
