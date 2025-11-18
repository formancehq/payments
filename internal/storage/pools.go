package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type pool struct {
	bun.BaseModel `bun:"table:pools"`

	// Mandatory fields
	ID        uuid.UUID       `bun:"id,pk,type:uuid,notnull"`
	Name      string          `bun:"name,type:text,notnull"`
	CreatedAt time.Time       `bun:"created_at,type:timestamp without time zone,notnull"`
	Type      models.PoolType `bun:"type,type:text,notnull"`
	Query     map[string]any  `bun:"query,type:jsonb,nullzero"`

	PoolAccounts []*poolAccounts `bun:"rel:has-many,join:id=pool_id"`
}

type poolAccounts struct {
	bun.BaseModel `bun:"table:pool_accounts"`

	PoolID      uuid.UUID          `bun:"pool_id,pk,type:uuid,notnull"`
	AccountID   models.AccountID   `bun:"account_id,pk,type:character varying,notnull"`
	ConnectorID models.ConnectorID `bun:"connector_id,type:character varying,notnull"`
}

func (s *store) PoolsUpsert(ctx context.Context, p models.Pool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return e("begin transaction: %w", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	poolToInsert, accountsToInsert := fromPoolModel(p)

	for i := range accountsToInsert {
		var exists bool
		exists, err = tx.NewSelect().
			Model((*account)(nil)).
			Where("id = ?", accountsToInsert[i].AccountID).
			Limit(1).
			Exists(ctx)
		if err != nil {
			return e("check account exists: %w", err)
		}

		if !exists {
			err = ErrNotFound // We need to define err here so that the rollback happens!
			return e("account does not exist: %w", err)
		}
	}

	var poolExists bool
	poolExists, err = tx.NewSelect().
		Model((*pool)(nil)).
		Where("id = ?", poolToInsert.ID).
		Exists(ctx)
	if err != nil {
		return e("check pool exists: %w", err)
	}

	var poolInsertRes sql.Result
	poolInsertRes, err = tx.NewInsert().
		Model(&poolToInsert).
		On("CONFLICT (id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return e("insert pool: %w", err)
	}

	var poolRowsAffected int64
	poolRowsAffected, err = poolInsertRes.RowsAffected()
	if err != nil {
		return e("get pool insert rows affected: %w", err)
	}

	if len(accountsToInsert) > 0 {
		_, err = tx.NewInsert().
			Model(&accountsToInsert).
			On("CONFLICT (pool_id, account_id) DO NOTHING").
			Exec(ctx)
		if err != nil {
			return e("insert pool accounts: %w", err)
		}
	}

	// Create outbox event only if pool was newly created (not updated)
	if !poolExists && poolRowsAffected > 0 {
		// Build account IDs array
		accountIDs := make([]string, len(p.PoolAccounts))
		for i := range p.PoolAccounts {
			accountIDs[i] = p.PoolAccounts[i].String()
		}

		payload := map[string]interface{}{
			"id":         p.ID.String(),
			"name":       p.Name,
			"createdAt":  p.CreatedAt,
			"accountIDs": accountIDs,
		}

		var payloadBytes []byte
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return e("failed to marshal pool event payload: %w", err)
		}

		outboxEvent := models.OutboxEvent{
			EventType:      models.OUTBOX_EVENT_POOL_SAVED,
			EntityID:       p.ID.String(),
			Payload:        payloadBytes,
			CreatedAt:      time.Now().UTC(),
			Status:         models.OUTBOX_STATUS_PENDING,
			ConnectorID:    nil, // Pools don't have connector ID
			IdempotencyKey: p.IdempotencyKey(),
		}

		if err = s.OutboxEventsInsert(ctx, tx, []models.OutboxEvent{outboxEvent}); err != nil {
			//if err = s.OutboxEventsInsertWithTx(ctx, []models.OutboxEvent{outboxEvent}); err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return e("commit transaction: %w", err)
	}
	return nil
}

func (s *store) PoolsGet(ctx context.Context, id uuid.UUID) (*models.Pool, error) {
	var pool pool
	err := s.db.NewSelect().
		Model(&pool).
		Relation("PoolAccounts").
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, e("get pool: %w", err)
	}

	return pointer.For(toPoolModel(pool)), nil
}

func (s *store) PoolsDelete(ctx context.Context, id uuid.UUID) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, e("begin transaction: %w", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	var res sql.Result
	res, err = tx.NewDelete().
		Model((*pool)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return false, e("delete pool: %w", err)
	}

	_, err = tx.NewDelete().
		Model((*poolAccounts)(nil)).
		Where("pool_id = ?", id).
		Exec(ctx)
	if err != nil {
		return false, e("delete pool accounts: %w", err)
	}

	var rowsAffected int64
	rowsAffected, err = res.RowsAffected()
	if err != nil {
		return false, e("get rows affected: %w", err)
	}

	// Create outbox event for pool deletion if pool was actually deleted
	if rowsAffected > 0 {
		payload := map[string]interface{}{
			"createdAt": time.Now().UTC(),
			"id":        id.String(),
		}

		var payloadBytes []byte
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return false, e("failed to marshal pool deleted event payload: %w", err)
		}

		outboxEvent := models.OutboxEvent{
			EventType:      models.OUTBOX_EVENT_POOL_DELETED,
			EntityID:       id.String(),
			Payload:        payloadBytes,
			CreatedAt:      time.Now().UTC(),
			Status:         models.OUTBOX_STATUS_PENDING,
			ConnectorID:    nil, // Pools don't have connector ID
			IdempotencyKey: id.String(),
		}

		if err = s.OutboxEventsInsert(ctx, tx, []models.OutboxEvent{outboxEvent}); err != nil {
			return false, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return false, e("commit transaction: %w", err)
	}
	return rowsAffected > 0, nil
}

func (s *store) PoolsAddAccount(ctx context.Context, id uuid.UUID, accountID models.AccountID) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return e("begin transaction: %w", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	var exists bool
	exists, err = tx.NewSelect().
		Model((*account)(nil)).
		Where("id = ?", accountID).
		Limit(1).
		Exists(ctx)
	if err != nil {
		return e("check account exists: %w", err)
	}

	if !exists {
		err = ErrNotFound // We need to define err here so that the rollback happens!
		return e("account does not exist: %w", err)
	}

	_, err = tx.NewInsert().
		Model(&poolAccounts{
			PoolID:      id,
			AccountID:   accountID,
			ConnectorID: accountID.ConnectorID,
		}).
		On("CONFLICT (pool_id, account_id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return e("insert pool account: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return e("commit transaction: %w", err)
	}
	return nil
}

func (s *store) PoolsRemoveAccount(ctx context.Context, id uuid.UUID, accountID models.AccountID) error {
	_, err := s.db.NewDelete().
		Model((*poolAccounts)(nil)).
		Where("pool_id = ? AND account_id = ?", id, accountID).
		Exec(ctx)
	if err != nil {
		return e("delete pool account: %w", err)
	}
	return nil
}

func (s *store) PoolsRemoveAccountsFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*poolAccounts)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)
	if err != nil {
		return e("delete pool accounts: %w", err)
	}
	return nil
}

type PoolQuery struct{}

type ListPoolsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PoolQuery]]

func NewListPoolsQuery(opts bunpaginate.PaginatedQueryOptions[PoolQuery]) ListPoolsQuery {
	return ListPoolsQuery{
		Order:    bunpaginate.OrderAsc,
		PageSize: opts.PageSize,
		Options:  opts,
	}
}

func (s *store) poolsQueryContext(qb query.Builder) (string, string, []any, error) {
	join := ""
	where, args, err := qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch key {
		case "name", "id":
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
			}

			return fmt.Sprintf("%s = ?", key), []any{value}, nil
		case "account_id":
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
			}

			join = "JOIN pool_accounts AS pool_accounts ON pool_accounts.pool_id = pool.id"

			return fmt.Sprintf("pool_accounts.%s = ?", key), []any{value}, nil
		default:
			return "", nil, fmt.Errorf("unknown key '%s' when building query: %w", key, ErrValidation)
		}
	}))

	return join, where, args, err
}

func (s *store) PoolsList(ctx context.Context, q ListPoolsQuery) (*bunpaginate.Cursor[models.Pool], error) {
	var (
		join  string
		where string
		args  []any
		err   error
	)
	if q.Options.QueryBuilder != nil {
		join, where, args, err = s.poolsQueryContext(q.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[PoolQuery], pool](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PoolQuery]])(&q),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			query = query.
				Relation("PoolAccounts")

			if join != "" {
				query = query.Join(join)
			}

			if where != "" {
				query = query.Where(where, args...)
			}

			query = query.Order("pool.created_at DESC", "pool.sort_id DESC")

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch pools", err)
	}

	pools := make([]models.Pool, 0, len(cursor.Data))
	for _, p := range cursor.Data {
		pools = append(pools, toPoolModel(p))
	}

	return &bunpaginate.Cursor[models.Pool]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     pools,
	}, nil
}

func fromPoolModel(from models.Pool) (pool, []poolAccounts) {
	p := pool{
		ID:        from.ID,
		Name:      from.Name,
		CreatedAt: from.CreatedAt,
		Type:      from.Type,
		Query:     from.Query,
	}

	var accounts []poolAccounts
	for i := range from.PoolAccounts {
		accounts = append(accounts, poolAccounts{
			PoolID:      from.ID,
			AccountID:   from.PoolAccounts[i],
			ConnectorID: from.PoolAccounts[i].ConnectorID,
		})
	}

	return p, accounts
}

func toPoolModel(from pool) models.Pool {
	var accounts []models.AccountID
	for i := range from.PoolAccounts {
		accounts = append(accounts, from.PoolAccounts[i].AccountID)
	}

	return models.Pool{
		ID:           from.ID,
		Name:         from.Name,
		CreatedAt:    from.CreatedAt,
		Type:         from.Type,
		Query:        from.Query,
		PoolAccounts: accounts,
	}
}
