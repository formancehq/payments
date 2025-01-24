package migrations

import (
	"context"
	"time"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/payments/internal/models"
	"github.com/uptrace/bun"
)

type v2Accounts struct {
	bun.BaseModel `bun:"accounts.account"`

	ID        models.AccountID `bun:"id,pk,type:character varying,nullzero"`
	CreatedAt time.Time        `bun:"created_at,type:timestamp with time zone,notnull"`
}

func MigrateAccountEventsFromV2(ctx context.Context, logger logging.Logger, db bun.IDB) error {
	exist, err := isTableExisting(ctx, db, "accounts", "account")
	if err != nil {
		return err
	}

	if !exist {
		// Nothing to migrate
		return nil
	}

	_, err = db.ExecContext(ctx, `
		ALTER TABLE accounts.account ADD COLUMN IF NOT EXISTS sort_id bigserial;
	`)
	if err != nil {
		return err
	}

	q := bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[any]]{
		Order:    bunpaginate.OrderAsc,
		PageSize: 1000,
		Options: bunpaginate.PaginatedQueryOptions[any]{
			PageSize: 1000,
		},
	}
	for {
		cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[any], v2Accounts](
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

		logger.WithField("accounts", len(cursor.Data)).Info("migrating accounts batch...")

		events := make([]v3eventSent, 0, len(cursor.Data))
		for _, account := range cursor.Data {
			events = append(events, v3eventSent{
				ID: models.EventID{
					EventIdempotencyKey: models.IdempotencyKey(account.ID),
					ConnectorID:         &account.ID.ConnectorID,
				},
				ConnectorID: &account.ID.ConnectorID,
				SentAt:      account.CreatedAt.UTC(),
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

		logger.WithField("accounts", len(cursor.Data)).Info("finished migrating accounts batch")

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
