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

type v2TransferInitiationAdjustment struct {
	bun.BaseModel `bun:"transfers.transfer_initiation_adjustments"`

	ID                   uuid.UUID                  `bun:"id,pk"`
	TransferInitiationID models.PaymentInitiationID `bun:"transfer_initiation_id"`
	CreatedAt            time.Time                  `bun:"created_at,nullzero"`
	Status               string                     `bun:"status"`
	Error                string                     `bun:"error"`
	Metadata             map[string]string          `bun:"metadata"`
}

type v3PaymentInitiationAdjustment struct {
	bun.BaseModel `bun:"payment_initiation_adjustments"`

	// Mandatory fields
	ID                  models.PaymentInitiationAdjustmentID     `bun:"id,pk,type:character varying,notnull"`
	PaymentInitiationID models.PaymentInitiationID               `bun:"payment_initiation_id,type:character varying,notnull"`
	CreatedAt           time.Time                                `bun:"created_at,type:timestamp without time zone,notnull"`
	Status              models.PaymentInitiationAdjustmentStatus `bun:"status,type:text,notnull"`

	// Optional fields
	Error  *string  `bun:"error,type:text"`
	Amount *big.Int `bun:"amount,type:numeric"`
	Asset  *string  `bun:"asset,type:text"`

	// Optional fields with default
	// c.f. https://bun.uptrace.dev/guide/models.html#default
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`
}

func MigrateTransferInitiationAdjustmentsFromV2(ctx context.Context, db bun.IDB) error {
	exist, err := isTableExisting(ctx, db, "transfers", "transfer_initiation_adjustments")
	if err != nil {
		return err
	}

	if !exist {
		// Nothing to migrate
		return nil
	}

	_, err = db.ExecContext(ctx, `
		ALTER TABLE transfers.transfer_initiation_adjustments ADD COLUMN IF NOT EXISTS sort_id bigserial;
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
		cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[any], v2TransferInitiationAdjustment](
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

		v3Adjs := make([]v3PaymentInitiationAdjustment, 0, len(cursor.Data))
		for _, adjustment := range cursor.Data {
			status, err := models.PaymentInitiationAdjustmentStatusFromString(adjustment.Status)
			if err != nil {
				// Some status disappeared in v3, let's skip them
				continue
			}

			amount := big.NewInt(0)
			asset := ""
			err = db.NewRaw(`SELECT amount, asset
				FROM transfers.transfer_initiation WHERE id = ?`, adjustment.TransferInitiationID).
				Scan(ctx, &amount, &asset)
			if err != nil {
				return err
			}

			v3Adjs = append(v3Adjs, v3PaymentInitiationAdjustment{
				ID: models.PaymentInitiationAdjustmentID{
					PaymentInitiationID: adjustment.TransferInitiationID,
					CreatedAt:           adjustment.CreatedAt,
					Status:              status,
				},
				PaymentInitiationID: adjustment.TransferInitiationID,
				CreatedAt:           adjustment.CreatedAt,
				Status:              status,
				Error:               &adjustment.Error,
				Amount:              amount,
				Asset:               &asset,
				Metadata:            adjustment.Metadata,
			})
		}

		if len(v3Adjs) > 0 {
			_, err = db.NewInsert().
				Model(&v3Adjs).
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
