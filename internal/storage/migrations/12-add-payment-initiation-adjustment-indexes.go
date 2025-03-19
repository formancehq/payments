package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func AddPaymentInitiationAdjustmentsIndexes(ctx context.Context, db bun.IDB) error {
	_, err := db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiation_adjustments_pi_id ON payment_initiation_adjustments (payment_initiation_id);
	`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiation_adjustments_sort_id ON payment_initiation_adjustments (sort_id);
	`)
	if err != nil {
		return err
	}
	return nil
}
