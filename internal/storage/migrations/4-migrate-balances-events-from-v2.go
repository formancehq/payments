package migrations

import (
	"context"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/uptrace/bun"
)

type v2Balance struct {
	bun.BaseModel `bun:"accounts.balances"`

	AccountID     models.AccountID `bun:"account_id"`
	Asset         string           `bun:"currency"`
	Balance       *big.Int         `bun:"balance"`
	CreatedAt     time.Time        `bun:"created_at"`
	LastUpdatedAt time.Time        `bun:"last_updated_at"`
}

func MigrateBalancesFromV2(ctx context.Context, db bun.IDB) error {
	exist, err := isTableExisting(ctx, db, "accounts", "balances")
	if err != nil {
		return err
	}

	if !exist {
		// Nothing to migrate
		return nil
	}

	_, err = db.ExecContext(ctx, `
		ALTER TABLE accounts.balances ADD COLUMN IF NOT EXISTS sort_id bigserial;
	`)
	if err != nil {
		return err
	}

	q := bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[any]]{
		Order:    bunpaginate.OrderAsc,
		PageSize: 100,
		Options: bunpaginate.PaginatedQueryOptions[any]{
			PageSize: 100,
		},
	}
	for {
		cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[any], v2Balance](
			ctx,
			db,
			(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[any]])(&q),
			func(query *bun.SelectQuery) *bun.SelectQuery {
				return query.Order("created_at ASC", "sort_id ASC")
			},
		)
		if err != nil {
			return err
		}

		events := make([]v3eventSent, 0, len(cursor.Data))
		for _, balance := range cursor.Data {
			b := models.Balance{
				AccountID:     balance.AccountID,
				CreatedAt:     balance.CreatedAt.UTC(),
				LastUpdatedAt: balance.LastUpdatedAt.UTC(),
				Asset:         balance.Asset,
				Balance:       balance.Balance,
			}

			events = append(events, v3eventSent{
				ID: models.EventID{
					EventIdempotencyKey: b.IdempotencyKey(),
					ConnectorID:         &balance.AccountID.ConnectorID,
				},
				ConnectorID: &balance.AccountID.ConnectorID,
				SentAt:      balance.LastUpdatedAt.UTC(),
			})
		}

		if len(events) > 0 {
			_, err = db.NewInsert().
				Model(&events).
				On("conflict (id) do nothing").
				Exec(ctx)
			if err != nil {
				return err
			}
		}

		if !cursor.HasMore {
			break
		}

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		if err != nil {
			return err
		}
	}

	return nil
}
