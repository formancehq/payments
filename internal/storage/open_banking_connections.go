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

type openBankingConnectionAttempt struct {
	bun.BaseModel `bun:"table:open_banking_connection_attempts"`

	// Mandatory fields
	ID          uuid.UUID                                 `bun:"id,pk,type:uuid,notnull"`
	PsuID       uuid.UUID                                 `bun:"psu_id,type:uuid,notnull"`
	ConnectorID models.ConnectorID                        `bun:"connector_id,type:character varying,notnull"`
	CreatedAt   time.Time                                 `bun:"created_at,type:timestamp without time zone,notnull"`
	Status      models.OpenBankingConnectionAttemptStatus `bun:"status,type:text,notnull"`
	State       json.RawMessage                           `bun:"state,type:jsonb,nullzero"`

	// Optional fields
	ClientRedirectURL *string    `bun:"client_redirect_url,type:text,nullzero"`
	TemporaryToken    *string    `bun:"temporary_token,type:text,nullzero"`
	ExpiresAt         *time.Time `bun:"expires_at,type:timestamp without time zone,nullzero"`
	Error             *string    `bun:"error,type:text,nullzero"`
}

func (s *store) OpenBankingConnectionAttemptsUpsert(ctx context.Context, from models.OpenBankingConnectionAttempt) error {
	attempt, err := fromOpenBankingConnectionAttemptsModels(from)
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

func (s *store) OpenBankingConnectionAttemptsUpdateStatus(ctx context.Context, id uuid.UUID, status models.OpenBankingConnectionAttemptStatus, errMsg *string) error {
	_, err := s.db.NewUpdate().
		Model((*openBankingConnectionAttempt)(nil)).
		Set("status = ?", status).
		Set("error = ?", errMsg).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return e("updating open banking connection attempt status", err)
	}

	return nil
}

func (s *store) OpenBankingConnectionAttemptsGet(ctx context.Context, id uuid.UUID) (*models.OpenBankingConnectionAttempt, error) {
	attempt := openBankingConnectionAttempt{}
	err := s.db.NewSelect().
		Model(&attempt).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, e("getting open banking connection attempt", err)
	}

	return toOpenBankingConnectionAttemptsModels(attempt)
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

func (s *store) OpenBankingConnectionAttemptsList(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, query ListPSUOpenBankingConnectionAttemptsQuery) (*bunpaginate.Cursor[models.OpenBankingConnectionAttempt], error) {
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

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[PSUOpenBankingConnectionAttemptsQuery], openBankingConnectionAttempt](s, ctx,
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
		return nil, e("failed to fetch open banking connection attempts", err)
	}

	psuOpenBankingConnectionAttemptsModels := make([]models.OpenBankingConnectionAttempt, len(cursor.Data))
	for i, attempt := range cursor.Data {
		res, err := toOpenBankingConnectionAttemptsModels(attempt)
		if err != nil {
			return nil, e("failed to fetch open banking connection attempts", err)
		}
		psuOpenBankingConnectionAttemptsModels[i] = *res
	}

	return &bunpaginate.Cursor[models.OpenBankingConnectionAttempt]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     psuOpenBankingConnectionAttemptsModels,
	}, nil
}

type openBankingForwardedUser struct {
	bun.BaseModel `bun:"table:open_banking_forwarded_users,alias:open_banking_forwarded_users"`

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

func (s *store) OpenBankingForwardedUserUpsert(ctx context.Context, psuID uuid.UUID, from models.OpenBankingForwardedUser) error {
	openBanking, token := fromOpenBankingForwardedUserModels(from, psuID)

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
			return e("upserting open banking forwarded user access token", err)
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
		return e("upserting open banking forwarded user", err)
	}

	return e("failed to commit transactions", tx.Commit())
}

func (s *store) OpenBankingForwardedUserGet(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) (*models.OpenBankingForwardedUser, error) {
	openBanking := openBankingForwardedUser{}
	err := s.db.NewSelect().
		Model(&openBanking).
		Column("open_banking_forwarded_users.*", "open_banking_access_tokens.access_token", "open_banking_access_tokens.expires_at").
		Join("LEFT JOIN open_banking_access_tokens ON open_banking_forwarded_users.psu_id = open_banking_access_tokens.psu_id AND open_banking_forwarded_users.connector_id = open_banking_access_tokens.connector_id").
		Where("open_banking_forwarded_users.psu_id = ?", psuID).
		Where("open_banking_forwarded_users.connector_id = ?", connectorID).
		Scan(ctx)

	if err != nil {
		return nil, e("getting open banking forwarded user", err)
	}

	return toOpenBankingForwardedUserModels(openBanking), nil
}

func (s *store) OpenBankingForwardedUserGetByPSPUserID(ctx context.Context, pspUserID string, connectorID models.ConnectorID) (*models.OpenBankingForwardedUser, error) {
	openBankingPSU := openBankingForwardedUser{}

	err := s.db.NewSelect().
		Model(&openBankingPSU).
		Column("open_banking_forwarded_users.*", "open_banking_access_tokens.access_token", "open_banking_access_tokens.expires_at").
		Join("LEFT JOIN open_banking_access_tokens ON open_banking_forwarded_users.psu_id = open_banking_access_tokens.psu_id AND open_banking_forwarded_users.connector_id = open_banking_access_tokens.connector_id").
		Where("open_banking_forwarded_users.psp_user_id = ?", pspUserID).
		Where("open_banking_forwarded_users.connector_id = ?", connectorID).
		Scan(ctx)

	if err != nil {
		return nil, e("getting open banking forwarded user", err)
	}

	return toOpenBankingForwardedUserModels(openBankingPSU), nil
}

func (s *store) OpenBankingForwardedUserDelete(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*openBankingForwardedUser)(nil)).
		Where("psu_id = ?", psuID).
		Where("connector_id = ?", connectorID).
		Exec(ctx)
	if err != nil {
		return e("deleting open banking forwarded user", err)
	}

	return nil
}

type OpenBankingForwardedUserQuery struct{}

type ListOpenBankingForwardedUserQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[OpenBankingForwardedUserQuery]]

func NewListOpenBankingForwardedUserQuery(opts bunpaginate.PaginatedQueryOptions[OpenBankingForwardedUserQuery]) ListOpenBankingForwardedUserQuery {
	return ListOpenBankingForwardedUserQuery{
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
			return fmt.Sprintf("open_banking_forwarded_users.%s = ?", key), []any{value}, nil
		case metadataRegex.Match([]byte(key)):
			if operator != "$match" {
				return "", nil, fmt.Errorf("'metadata' column can only be used with $match: %w", ErrValidation)
			}
			match := metadataRegex.FindAllStringSubmatch(key, 3)

			key := "open_banking_forwarded_users.metadata"
			return key + " @> ?", []any{map[string]any{
				match[0][1]: value,
			}}, nil
		default:
			return "", nil, fmt.Errorf("unknown key '%s' when building query: %w", key, ErrValidation)
		}
	}))
}

func (s *store) OpenBankingForwardedUserList(ctx context.Context, query ListOpenBankingForwardedUserQuery) (*bunpaginate.Cursor[models.OpenBankingForwardedUser], error) {
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

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[OpenBankingForwardedUserQuery], openBankingForwardedUser](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[OpenBankingForwardedUserQuery]])(&query),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			if where != "" {
				query = query.Where(where, args...)
			}

			query = query.
				Column("open_banking_forwarded_users.*", "open_banking_access_tokens.access_token", "open_banking_access_tokens.expires_at").
				Join("LEFT JOIN open_banking_access_tokens ON open_banking_forwarded_users.psu_id = open_banking_access_tokens.psu_id AND open_banking_forwarded_users.connector_id = open_banking_access_tokens.connector_id")

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch open banking forwarded user", err)
	}

	psuOpenBankingModels := make([]models.OpenBankingForwardedUser, len(cursor.Data))
	for i, p := range cursor.Data {
		bb := toOpenBankingForwardedUserModels(p)
		psuOpenBankingModels[i] = *bb
	}

	return &bunpaginate.Cursor[models.OpenBankingForwardedUser]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     psuOpenBankingModels,
	}, nil
}

type openBankingConnections struct {
	bun.BaseModel `bun:"table:open_banking_connections"`

	// Mandatory fields
	PsuID         uuid.UUID               `bun:"psu_id,pk,type:uuid,notnull"`
	ConnectorID   models.ConnectorID      `bun:"connector_id,pk,type:character varying,notnull"`
	ConnectionID  string                  `bun:"connection_id,pk,type:character varying,notnull"`
	CreatedAt     time.Time               `bun:"created_at,type:timestamp without time zone,notnull"`
	DataUpdatedAt time.Time               `bun:"data_updated_at,type:timestamp without time zone,notnull"`
	Status        models.ConnectionStatus `bun:"status,type:text,notnull"`
	UpdatedAt     time.Time               `bun:"updated_at,type:timestamp without time zone,notnull"`

	// Optional fields
	Error    *string           `bun:"error,type:text,nullzero"`
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`

	// ScanOnly fields
	AccessToken *string    `bun:"access_token,type:text,nullzero,scanonly"`
	ExpiresAt   *time.Time `bun:"expires_at,type:timestamp without time zone,nullzero,scanonly"`
}

func (s *store) OpenBankingConnectionsUpsert(ctx context.Context, psuID uuid.UUID, from models.OpenBankingConnection) error {
	connection, token := fromOpenBankingConnectionsModels(from, psuID)

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
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return e("upserting open banking connection", err)
	}

	return e("failed to commit transactions", tx.Commit())
}

func (s *store) OpenBankingConnectionsUpdateLastDataUpdate(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string, updatedAt libtime.Time) error {
	_, err := s.db.NewUpdate().
		Model((*openBankingConnections)(nil)).
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

func (s *store) OpenBankingConnectionsGet(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string) (*models.OpenBankingConnection, error) {
	connection := openBankingConnections{}
	err := s.db.NewSelect().
		Model(&connection).
		Column("open_banking_connections.*", "open_banking_access_tokens.access_token", "open_banking_access_tokens.expires_at").
		Join("LEFT JOIN open_banking_access_tokens ON open_banking_connections.psu_id = open_banking_access_tokens.psu_id AND open_banking_connections.connector_id = open_banking_access_tokens.connector_id AND open_banking_connections.connection_id = open_banking_access_tokens.connection_id").
		Where("open_banking_connections.psu_id = ?", psuID).
		Where("open_banking_connections.connector_id = ?", connectorID).
		Where("open_banking_connections.connection_id = ?", connectionID).
		Scan(ctx)
	if err != nil {
		return nil, e("getting open banking connection", err)
	}

	return pointer.For(toOpenBankingConnectionsModels(connection)), nil
}

func (s *store) OpenBankingConnectionsGetFromConnectionID(ctx context.Context, connectorID models.ConnectorID, connectionID string) (*models.OpenBankingConnection, uuid.UUID, error) {
	connection := openBankingConnections{}
	err := s.db.NewSelect().
		Model(&connection).
		Column("open_banking_connections.*", "open_banking_access_tokens.access_token", "open_banking_access_tokens.expires_at").
		Join("LEFT JOIN open_banking_access_tokens ON open_banking_connections.psu_id = open_banking_access_tokens.psu_id AND open_banking_connections.connector_id = open_banking_access_tokens.connector_id AND open_banking_connections.connection_id = open_banking_access_tokens.connection_id").
		Where("open_banking_connections.connector_id = ?", connectorID).
		Where("open_banking_connections.connection_id = ?", connectionID).
		Scan(ctx)
	if err != nil {
		return nil, uuid.Nil, e("getting open banking connection", err)
	}

	return pointer.For(toOpenBankingConnectionsModels(connection)), connection.PsuID, nil
}

func (s *store) OpenBankingConnectionsDelete(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string) error {
	_, err := s.db.NewDelete().
		Model((*openBankingConnections)(nil)).
		Where("psu_id = ?", psuID).
		Where("connector_id = ?", connectorID).
		Where("connection_id = ?", connectionID).
		Exec(ctx)
	if err != nil {
		return e("deleting open banking connection", err)
	}

	return nil
}

type OpenBankingConnectionsQuery struct{}

type ListOpenBankingConnectionsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[OpenBankingConnectionsQuery]]

func NewListOpenBankingConnectionsQuery(opts bunpaginate.PaginatedQueryOptions[OpenBankingConnectionsQuery]) ListOpenBankingConnectionsQuery {
	return ListOpenBankingConnectionsQuery{
		PageSize: opts.PageSize,
		Order:    bunpaginate.OrderAsc,
		Options:  opts,
	}
}

func (s *store) openBankingConnectionsQueryContext(qb query.Builder) (string, []any, error) {
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

func (s *store) OpenBankingConnectionsList(ctx context.Context, psuID uuid.UUID, connectorID *models.ConnectorID, query ListOpenBankingConnectionsQuery) (*bunpaginate.Cursor[models.OpenBankingConnection], error) {
	var (
		where string
		args  []any
		err   error
	)
	if query.Options.QueryBuilder != nil {
		where, args, err = s.openBankingConnectionsQueryContext(query.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[OpenBankingConnectionsQuery], openBankingConnections](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[OpenBankingConnectionsQuery]])(&query),
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
		return nil, e("failed to fetch open banking connections", err)
	}

	openBankingConnections := make([]openBankingConnections, len(cursor.Data))
	copy(openBankingConnections, cursor.Data)

	openBankingConnectionsModels := make([]models.OpenBankingConnection, len(openBankingConnections))
	for i, connection := range openBankingConnections {
		openBankingConnectionsModels[i] = toOpenBankingConnectionsModels(connection)
	}

	return &bunpaginate.Cursor[models.OpenBankingConnection]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     openBankingConnectionsModels,
	}, nil
}

type openBankingAccessTokens struct {
	bun.BaseModel `bun:"table:open_banking_access_tokens"`

	// Mandatory fields
	PSUID        uuid.UUID          `bun:"psu_id,type:uuid,notnull"`
	ConnectorID  models.ConnectorID `bun:"connector_id,type:character varying,notnull"`
	ConnectionID *string            `bun:"connection_id,type:character varying,nullzero"`
	AccessToken  string             `bun:"access_token,type:text,notnull"`
	ExpiresAt    time.Time          `bun:"expires_at,type:timestamp without time zone,notnull"`
}

func fromOpenBankingConnectionAttemptsModels(from models.OpenBankingConnectionAttempt) (openBankingConnectionAttempt, error) {
	var token *string
	var expiresAt *time.Time
	if from.TemporaryToken != nil {
		t, e := fromTokenModels(*from.TemporaryToken)
		token = &t
		expiresAt = &e
	}

	state, err := json.Marshal(from.State)
	if err != nil {
		return openBankingConnectionAttempt{}, err
	}

	return openBankingConnectionAttempt{
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

func toOpenBankingConnectionAttemptsModels(from openBankingConnectionAttempt) (*models.OpenBankingConnectionAttempt, error) {
	state := models.CallbackState{}
	if err := json.Unmarshal(from.State, &state); err != nil {
		return nil, err
	}

	return &models.OpenBankingConnectionAttempt{
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

func fromOpenBankingForwardedUserModels(from models.OpenBankingForwardedUser, psuID uuid.UUID) (openBankingForwardedUser, *openBankingAccessTokens) {
	var token *openBankingAccessTokens
	if from.AccessToken != nil {
		accessToken, expiresAt := fromTokenModels(*from.AccessToken)

		token = &openBankingAccessTokens{
			PSUID:       psuID,
			ConnectorID: from.ConnectorID,
			AccessToken: accessToken,
			ExpiresAt:   expiresAt,
		}
	}

	return openBankingForwardedUser{
		PsuID:       psuID,
		PSPUserID:   from.PSPUserID,
		ConnectorID: from.ConnectorID,
		Metadata:    from.Metadata,
	}, token
}

func toOpenBankingForwardedUserModels(from openBankingForwardedUser) *models.OpenBankingForwardedUser {
	return &models.OpenBankingForwardedUser{
		PsuID:       from.PsuID,
		ConnectorID: from.ConnectorID,
		PSPUserID:   from.PSPUserID,
		AccessToken: toTokenModels(from.AccessToken, from.ExpiresAt),
		Metadata:    from.Metadata,
	}
}

func fromOpenBankingConnectionsModels(from models.OpenBankingConnection, psuID uuid.UUID) (openBankingConnections, *openBankingAccessTokens) {
	var token *openBankingAccessTokens
	if from.AccessToken != nil {
		accessToken, expiresAt := fromTokenModels(*from.AccessToken)

		token = &openBankingAccessTokens{
			PSUID:        psuID,
			ConnectorID:  from.ConnectorID,
			ConnectionID: &from.ConnectionID,
			AccessToken:  accessToken,
			ExpiresAt:    expiresAt,
		}
	}

	return openBankingConnections{
		PsuID:         psuID,
		ConnectorID:   from.ConnectorID,
		ConnectionID:  from.ConnectionID,
		CreatedAt:     time.New(from.CreatedAt),
		DataUpdatedAt: time.New(from.DataUpdatedAt),
		Status:        from.Status,
		UpdatedAt:     time.New(from.UpdatedAt),
		Error:         from.Error,
		Metadata:      from.Metadata,
	}, token
}

func toOpenBankingConnectionsModels(from openBankingConnections) models.OpenBankingConnection {
	return models.OpenBankingConnection{
		ConnectorID:   from.ConnectorID,
		ConnectionID:  from.ConnectionID,
		CreatedAt:     from.CreatedAt.Time,
		DataUpdatedAt: from.DataUpdatedAt.Time,
		Status:        from.Status,
		UpdatedAt:     from.UpdatedAt.Time,
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
