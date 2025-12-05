package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	internalTime "github.com/formancehq/go-libs/v3/time"
	internalEvents "github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

type balance struct {
	bun.BaseModel `bun:"table:balances"`

	// Mandatory fields
	AccountID models.AccountID  `bun:"account_id,pk,type:character varying,notnull"`
	CreatedAt internalTime.Time `bun:"created_at,pk,type:timestamp without time zone,notnull"`
	Asset     string            `bun:"asset,pk,type:text,notnull"`

	ConnectorID   models.ConnectorID `bun:"connector_id,type:character varying,notnull"`
	Balance       *big.Int           `bun:"balance,type:numeric,notnull"`
	LastUpdatedAt internalTime.Time  `bun:"last_updated_at,type:timestamp without time zone,notnull"`

	// Optional fields
	PsuID                   *uuid.UUID `bun:"psu_id,type:uuid,nullzero"`
	OpenBankingConnectionID *string    `bun:"open_banking_connection_id,type:character varying,nullzero"`
}

func (s *store) BalancesUpsert(ctx context.Context, balances []models.Balance) error {
	toInsert := fromBalancesModels(balances)
	if len(balances) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	// Track newly inserted/updated balances for outbox events
	var insertedBalances []models.Balance

	var changed bool
	for i, balance := range toInsert {
		changed, err = s.insertBalances(ctx, tx, &balance)
		if err != nil {
			return err
		}
		// Only append to insertedBalances if the DB was actually modified
		if changed {
			insertedBalances = append(insertedBalances, balances[i])
		}
	}

	// Create outbox events for all balances
	if len(insertedBalances) > 0 {
		outboxEvents := make([]models.OutboxEvent, 0, len(insertedBalances))
		for _, balance := range insertedBalances {
			// Create the event payload
			payload := internalEvents.BalanceMessagePayload{
				AccountID:     balance.AccountID.String(),
				ConnectorID:   balance.AccountID.ConnectorID.String(),
				Provider:      models.ToV3Provider(balance.AccountID.ConnectorID.Provider),
				CreatedAt:     balance.CreatedAt,
				LastUpdatedAt: balance.LastUpdatedAt,
				Asset:         balance.Asset,
				Balance:       balance.Balance,
			}

			var payloadBytes []byte
			payloadBytes, err = json.Marshal(&payload)
			if err != nil {
				return e("failed to marshal balance event payload", err)
			}

			connectorID := balance.AccountID.ConnectorID
			outboxEvent := models.OutboxEvent{
				ID: models.EventID{
					EventIdempotencyKey: balance.IdempotencyKey(),
					ConnectorID:         &connectorID,
				},
				EventType:   events.EventTypeSavedBalances,
				EntityID:    balance.AccountID.String(),
				Payload:     payloadBytes,
				CreatedAt:   time.Now().UTC(),
				Status:      models.OUTBOX_STATUS_PENDING,
				ConnectorID: &connectorID,
			}

			outboxEvents = append(outboxEvents, outboxEvent)
		}

		// Insert outbox events in the same transaction
		if err = s.OutboxEventsInsert(ctx, tx, outboxEvents); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return e("failed to commit transaction", err)
	}
	return nil
}

func (s *store) insertBalances(ctx context.Context, tx bun.Tx, balance *balance) (bool, error) {
	var lastBalance models.Balance
	found := true
	err := tx.NewSelect().
		Model(&lastBalance).
		Where("account_id = ? AND asset = ?", balance.AccountID, balance.Asset).
		Order("created_at DESC", "sort_id DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		pErr := e("failed to get balance", err)
		if !errors.Is(pErr, ErrNotFound) {
			return false, pErr
		}
		found = false
	}

	if found && lastBalance.CreatedAt.After(balance.CreatedAt.Time) {
		// Do not insert balance if the last balance is newer
		return false, nil
	}

	switch {
	case found && lastBalance.Balance.Cmp(balance.Balance) == 0:
		// same balance, no need to have a new entry, just update the last one
		_, err = tx.NewUpdate().
			Model((*models.Balance)(nil)).
			Set("last_updated_at = ?", balance.LastUpdatedAt).
			Where("account_id = ? AND created_at = ? AND asset = ?", lastBalance.AccountID, lastBalance.CreatedAt, lastBalance.Asset).
			Exec(ctx)
		if err != nil {
			return false, e("failed to update balance", err)
		}
		return true, nil

	case found && lastBalance.Balance.Cmp(balance.Balance) != 0:
		// different balance, insert a new entry
		_, err = tx.NewInsert().
			Model(balance).
			Exec(ctx)
		if err != nil {
			return false, e("failed to insert balance", err)
		}

		// and update last row last updated at to this created at
		_, err = tx.NewUpdate().
			Model(&lastBalance).
			Set("last_updated_at = ?", balance.CreatedAt).
			Where("account_id = ? AND created_at = ? AND asset = ?", lastBalance.AccountID, lastBalance.CreatedAt, lastBalance.Asset).
			Exec(ctx)
		if err != nil {
			return false, e("failed to update balance", err)
		}
		return true, nil

	case !found:
		// no balance found, insert a new entry
		_, err = tx.NewInsert().
			Model(balance).
			Exec(ctx)
		if err != nil {
			return false, e("failed to insert balance", err)
		}
		return true, nil
	}

	return false, nil
}

func (s *store) BalancesDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*balance)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)
	if err != nil {
		return e("delete balances", err)
	}

	return nil
}

type BalanceQuery struct {
	AccountID *models.AccountID
	Asset     string
	From      time.Time
	To        time.Time
}

func NewBalanceQuery() BalanceQuery {
	return BalanceQuery{}
}

func (b BalanceQuery) WithAccountID(accountID *models.AccountID) BalanceQuery {
	b.AccountID = accountID

	return b
}

func (b BalanceQuery) WithAsset(asset string) BalanceQuery {
	b.Asset = asset

	return b
}

func (b BalanceQuery) WithFrom(from time.Time) BalanceQuery {
	b.From = from

	return b
}

func (b BalanceQuery) WithTo(to time.Time) BalanceQuery {
	b.To = to

	return b
}

func applyBalanceQuery(query *bun.SelectQuery, balanceQuery BalanceQuery) *bun.SelectQuery {
	if balanceQuery.AccountID != nil {
		query = query.Where("balance.account_id = ?", balanceQuery.AccountID)
	}

	if balanceQuery.Asset != "" {
		query = query.Where("balance.asset = ?", balanceQuery.Asset)
	}

	if !balanceQuery.From.IsZero() {
		query = query.Where("balance.last_updated_at >= ?", balanceQuery.From)
	}

	if !balanceQuery.To.IsZero() {
		query = query.Where("(balance.created_at <= ?)", balanceQuery.To)
	}

	return query
}

type ListBalancesQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[BalanceQuery]]

func NewListBalancesQuery(opts bunpaginate.PaginatedQueryOptions[BalanceQuery]) ListBalancesQuery {
	return ListBalancesQuery{
		Order:    bunpaginate.OrderAsc,
		PageSize: opts.PageSize,
		Options:  opts,
	}
}

func (s *store) BalancesList(ctx context.Context, q ListBalancesQuery) (*bunpaginate.Cursor[models.Balance], error) {
	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[BalanceQuery], balance](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[BalanceQuery]])(&q),
		func(query *bun.SelectQuery) *bun.SelectQuery {

			query = applyBalanceQuery(query, q.Options.Options)

			query = query.Order("created_at DESC", "sort_id DESC")

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch balances", err)
	}

	balances := toBalancesModels(cursor.Data)

	return &bunpaginate.Cursor[models.Balance]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     balances,
	}, nil
}

// Get balances from account IDs at a specific time. If at is nil, it will return the latest balances.
func (s *store) BalancesGetFromAccountIDs(ctx context.Context, accountIDs []models.AccountID, at *time.Time) ([]models.AggregatedBalance, error) {
	if len(accountIDs) == 0 {
		// return empty array if no account IDs are provided
		return []models.AggregatedBalance{}, nil
	}

	type assetBalances struct {
		AccountIDs []string `bun:"account_ids,array"`
		Asset      string   `bun:"asset"`
		Balance    *big.Int `bun:"balance"`
	}

	selectedBalancesQuery := s.db.NewSelect().
		Model((*balance)(nil)).
		DistinctOn("account_id, asset").
		Column("account_id", "asset", "created_at", "sort_id", "balance").
		Where("account_id IN (?)", bun.In(accountIDs)).
		Order("account_id desc", "asset desc", "created_at desc", "sort_id desc")

	if at != nil && !at.IsZero() {
		selectedBalancesQuery = selectedBalancesQuery.Where("created_at <= ?", at).
			Where("last_updated_at >= ? OR last_updated_at = created_at", at)
	}
	var balanceAssets []assetBalances
	query := s.db.NewSelect().
		With(
			"selected_balances",
			selectedBalancesQuery,
		).
		ModelTableExpr("selected_balances").
		ColumnExpr("array_agg(account_id) as account_ids, asset, SUM(balance) AS balance").
		Group("asset")

	err := query.Scan(ctx, &balanceAssets)
	if err != nil {
		return nil, e("failed to list balance assets", err)
	}

	res := make([]models.AggregatedBalance, 0, len(balanceAssets))
	for _, balanceAsset := range balanceAssets {
		relatedAccounts := make([]models.AccountID, len(balanceAsset.AccountIDs))
		for i, accountID := range balanceAsset.AccountIDs {
			relatedAccounts[i] = models.MustAccountIDFromString(accountID)
		}
		res = append(res, models.AggregatedBalance{
			Asset:           balanceAsset.Asset,
			Amount:          balanceAsset.Balance,
			RelatedAccounts: relatedAccounts,
		})
	}

	return res, nil
}

func fromBalancesModels(from []models.Balance) []balance {
	var to []balance
	for _, b := range from {
		to = append(to, fromBalanceModels(b))
	}
	return to
}

func fromBalanceModels(from models.Balance) balance {
	return balance{
		BaseModel:               bun.BaseModel{},
		AccountID:               from.AccountID,
		CreatedAt:               internalTime.New(from.CreatedAt),
		Asset:                   from.Asset,
		ConnectorID:             from.AccountID.ConnectorID,
		Balance:                 from.Balance,
		LastUpdatedAt:           internalTime.New(from.LastUpdatedAt),
		PsuID:                   from.PsuID,
		OpenBankingConnectionID: from.OpenBankingConnectionID,
	}
}

func toBalancesModels(from []balance) []models.Balance {
	to := make([]models.Balance, 0, len(from))
	for _, b := range from {
		to = append(to, toBalanceModels(b))
	}
	return to
}

func toBalanceModels(from balance) models.Balance {
	return models.Balance{
		AccountID:               from.AccountID,
		CreatedAt:               from.CreatedAt.Time,
		LastUpdatedAt:           from.LastUpdatedAt.Time,
		Asset:                   from.Asset,
		Balance:                 from.Balance,
		PsuID:                   from.PsuID,
		OpenBankingConnectionID: from.OpenBankingConnectionID,
	}
}
