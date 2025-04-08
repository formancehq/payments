package migrations

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/uptrace/bun"
)

type v2TransferInitiation struct {
	bun.BaseModel `bun:"transfers.transfer_initiation"`

	ID        models.PaymentInitiationID `bun:"id"`
	Reference string                     `bun:"reference"`
}

func FixMissingReferenceTransferInitiation(ctx context.Context, db bun.IDB) error {
	exist, err := isTableExisting(ctx, db, "transfers", "transfer_initiation")
	if err != nil {
		return err
	}

	if !exist {
		// Nothing to migrate
		return nil
	}

	_, err = db.ExecContext(ctx, `
		ALTER TABLE transfers.transfer_initiation ADD COLUMN IF NOT EXISTS reference text;
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		ALTER TABLE transfers.transfer_initiation ADD COLUMN IF NOT EXISTS sort_id bigserial;
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
		cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[any], v2TransferInitiation](
			ctx,
			db,
			(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[any]])(&q),
			func(query *bun.SelectQuery) *bun.SelectQuery {
				return query.
					Column("id").
					Order("created_at ASC", "sort_id ASC")
			},
		)
		if err != nil {
			return err
		}

		for i := range cursor.Data {
			cursor.Data[i].Reference = cursor.Data[i].ID.Reference

			_, err = db.NewUpdate().
				Model(&cursor.Data[i]).
				Set("reference = ?", cursor.Data[i].Reference).
				Where("id = ?", cursor.Data[i].ID).
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
