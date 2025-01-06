package migrations

import (
	"context"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type v2Payment struct {
	bun.BaseModel `bun:"payments.payment"`

	ID          models.PaymentID     `bun:"id,pk,type:character varying"`
	ConnectorID models.ConnectorID   `bun:"connector_id"`
	CreatedAt   time.Time            `bun:"created_at"`
	Reference   string               `bun:"reference"`
	Status      models.PaymentStatus `bun:"status"`
}

type v2PaymentAdjustments struct {
	bun.BaseModel `bun:"payments.adjustment"`

	ID        uuid.UUID            `bun:"id,pk,nullzero"`
	PaymentID models.PaymentID     `bun:"payment_id,pk,nullzero"`
	CreatedAt time.Time            `bun:"created_at,nullzero"`
	Reference string               `bun:"reference"`
	Amount    *big.Int             `bun:"amount"`
	Status    models.PaymentStatus `bun:"status"`
}

func MigratePaymentsAdjustmentsFromV2(ctx context.Context, db bun.IDB) error {
	exist, err := isTableExisting(ctx, db, "payments", "adjustment")
	if err != nil {
		return err
	}

	if !exist {
		// Nothing to migrate
		return nil
	}

	_, err = db.ExecContext(ctx, `
		ALTER TABLE payments.adjustment ADD COLUMN IF NOT EXISTS sort_id bigserial;
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
		cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[any], v2PaymentAdjustments](
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
		for _, adjustment := range cursor.Data {
			events = append(events, v3eventSent{
				ID: models.EventID{
					EventIdempotencyKey: models.IdempotencyKey(models.PaymentAdjustmentID{
						PaymentID: adjustment.PaymentID,
						Reference: adjustment.Reference,
						CreatedAt: adjustment.CreatedAt.UTC(),
						Status:    adjustment.Status,
					}),
					ConnectorID: &adjustment.PaymentID.ConnectorID,
				},
				ConnectorID: &adjustment.PaymentID.ConnectorID,
				SentAt:      adjustment.CreatedAt.UTC(),
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

// We also have to migrate the payments table from v2 to v3 because some
// connectors do not use the adjustments table.
func MigratePaymentsFromV2(ctx context.Context, db bun.IDB) error {
	exist, err := isTableExisting(ctx, db, "payments", "payment")
	if err != nil {
		return err
	}

	if !exist {
		// Nothing to migrate
		return nil
	}

	_, err = db.ExecContext(ctx, `
		ALTER TABLE payments.payment ADD COLUMN IF NOT EXISTS sort_id bigserial;
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
		cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[any], v2Payment](
			ctx,
			db,
			(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[any]])(&q),
			func(query *bun.SelectQuery) *bun.SelectQuery {
				return query.
					Column("id", "connector_id", "created_at", "reference", "status").
					Order("created_at ASC", "sort_id ASC")
			},
		)
		if err != nil {
			return err
		}

		events := make([]v3eventSent, 0, len(cursor.Data))
		for _, payment := range cursor.Data {
			events = append(events, v3eventSent{
				ID: models.EventID{
					EventIdempotencyKey: models.IdempotencyKey(models.PaymentAdjustmentID{
						PaymentID: payment.ID,
						Reference: payment.Reference,
						CreatedAt: payment.CreatedAt.UTC(),
						Status:    payment.Status,
					}),
					ConnectorID: &payment.ConnectorID,
				},
				ConnectorID: &payment.ConnectorID,
				SentAt:      payment.CreatedAt.UTC(),
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

type PaymentType string
