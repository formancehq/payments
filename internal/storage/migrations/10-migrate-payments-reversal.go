package migrations

import (
	"context"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

type v2TransferReversal struct {
	bun.BaseModel `bun:"transfers.transfer_reversal"`

	ID                   models.PaymentInitiationReversalID `bun:"id,pk"`
	TransferInitiationID models.PaymentInitiationID         `bun:"transfer_initiation_id"`

	CreatedAt   time.Time `bun:"created_at"`
	UpdatedAt   time.Time `bun:"updated_at"`
	Description string    `bun:"description"`

	ConnectorID models.ConnectorID `bun:"connector_id"`

	Amount *big.Int `bun:"amount"`
	Asset  string   `bun:"asset"`

	Status models.PaymentInitiationReversalAdjustmentStatus `bun:"status"`
	Error  string                                           `bun:"error"`

	Metadata map[string]string `bun:"metadata"`
}

type v3PaymentInitiationReversal struct {
	bun.BaseModel `bun:"payment_initiation_reversals"`

	ID                  models.PaymentInitiationReversalID `bun:"id,pk,type:character varying,notnull"`
	ConnectorID         models.ConnectorID                 `bun:"connector_id,type:character varying,notnull"`
	PaymentInitiationID models.PaymentInitiationID         `bun:"payment_initiation_id,type:character varying,notnull"`
	Reference           string                             `bun:"reference,type:text,notnull"`
	CreatedAt           time.Time                          `bun:"created_at,type:timestamp without time zone,notnull"`
	Description         string                             `bun:"description,type:text,notnull"`
	Amount              *big.Int                           `bun:"amount,type:numeric,notnull"`
	Asset               string                             `bun:"asset,type:text,notnull"`

	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`
}

type v3PaymentInitiationReversalAdjustment struct {
	bun.BaseModel `bun:"payment_initiation_reversal_adjustments"`

	ID                          models.PaymentInitiationReversalAdjustmentID `bun:"id,pk,type:character varying,notnull"`
	PaymentInitiationReversalID models.PaymentInitiationReversalID           `bun:"payment_initiation_reversal_id,type:character varying,notnull"`
	CreatedAt                   time.Time                                    `bun:"created_at,type:timestamp without time zone,notnull"`
	Status                      models.PaymentInitiationReversalAdjustmentStatus

	Error *string `bun:"error,type:text"`

	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`
}

func MigrateTransferReversalsFromV2(ctx context.Context, db bun.IDB) error {
	exist, err := isTableExisting(ctx, db, "transfers", "transfer_reversal")
	if err != nil {
		return err
	}

	if !exist {
		// Nothing to migrate
		return nil
	}

	_, err = db.ExecContext(ctx, `
		ALTER TABLE transfers.transfer_reversal ADD COLUMN IF NOT EXISTS sort_id bigserial;
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
		cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[any], v2TransferReversal](
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

		v3Reversals := make([]v3PaymentInitiationReversal, 0, len(cursor.Data))
		v3ReversalAdjustments := make([]v3PaymentInitiationReversalAdjustment, 0)
		for _, reversal := range cursor.Data {
			reversal.Status++ // needed as we added the unknown status as 0 in v3

			v3Reversals = append(v3Reversals, v3PaymentInitiationReversal{
				ID:                  reversal.ID,
				ConnectorID:         reversal.ConnectorID,
				PaymentInitiationID: reversal.TransferInitiationID,
				Reference:           reversal.ID.Reference,
				CreatedAt:           reversal.CreatedAt,
				Description:         reversal.Description,
				Amount:              reversal.Amount,
				Asset:               reversal.Asset,
				Metadata:            reversal.Metadata,
			})

			v3ReversalAdjustments = append(v3ReversalAdjustments, v3PaymentInitiationReversalAdjustment{
				ID: models.PaymentInitiationReversalAdjustmentID{
					PaymentInitiationReversalID: reversal.ID,
					CreatedAt:                   reversal.CreatedAt,
					Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSING,
				},
				PaymentInitiationReversalID: reversal.ID,
				CreatedAt:                   reversal.CreatedAt,
				Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSING,
				Metadata:                    reversal.Metadata,
			})

			if reversal.Status != models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSING {
				v3ReversalAdjustments = append(v3ReversalAdjustments, v3PaymentInitiationReversalAdjustment{
					BaseModel: schema.BaseModel{},
					ID: models.PaymentInitiationReversalAdjustmentID{
						PaymentInitiationReversalID: reversal.ID,
						CreatedAt:                   reversal.CreatedAt,
						Status:                      reversal.Status,
					},
					PaymentInitiationReversalID: reversal.ID,
					CreatedAt:                   reversal.CreatedAt,
					Status:                      reversal.Status,
					Error: func() *string {
						if reversal.Error == "" {
							return nil
						}

						return &reversal.Error
					}(),
					Metadata: reversal.Metadata,
				})
			}
		}

		if len(v3Reversals) > 0 {
			_, err = db.NewInsert().
				Model(&v3Reversals).
				On("conflict (id) do nothing").
				Exec(ctx)
			if err != nil {
				return err
			}
		}

		if len(v3ReversalAdjustments) > 0 {
			_, err = db.NewInsert().
				Model(&v3ReversalAdjustments).
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
