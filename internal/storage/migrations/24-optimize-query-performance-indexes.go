package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

// AddOptimizeQueryPerformanceIndexes adds comprehensive indexes to improve query performance across the application.
// All indexes use CREATE INDEX CONCURRENTLY for zero-downtime deployment.
// Each index is created in a separate ExecContext call to avoid transaction wrapping issues.
func AddOptimizeQueryPerformanceIndexes(ctx context.Context, db bun.IDB) error {
	// ============================================
	// JSONB GIN Indexes
	// ============================================
	// Dramatically improve metadata filtering performance (100x+ faster)

	_, err := db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_payments_metadata_gin
		    ON payments USING gin (metadata);
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_accounts_metadata_gin
		    ON accounts USING gin (metadata);
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_payment_adjustments_metadata_gin
		    ON payment_adjustments USING gin (metadata);
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_payment_initiations_metadata_gin
		    ON payment_initiations USING gin (metadata);
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_payment_initiation_adjustments_metadata_gin
		    ON payment_initiation_adjustments USING gin (metadata);
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_payment_initiation_reversals_metadata_gin
		    ON payment_initiation_reversals USING gin (metadata);
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_bank_accounts_metadata_gin
		    ON bank_accounts USING gin (metadata);
	`)
	if err != nil {
		return err
	}

	// ============================================
	// Payment Adjustment Optimizations
	// ============================================
	// Composite index for LATERAL JOIN in PaymentsList (payments.go:432-439)
	// Covers: WHERE payment_id = X ORDER BY created_at DESC, sort_id DESC LIMIT 1

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_payment_adjustments_payment_created_sort
		    ON payment_adjustments (payment_id, created_at DESC, sort_id DESC);
	`)
	if err != nil {
		return err
	}

	// ============================================
	// Reference Lookup Optimizations
	// ============================================
	// Composite index for PaymentsGetByReference (payments.go:254-264)

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_payments_connector_reference
		    ON payments (connector_id, reference);
	`)
	if err != nil {
		return err
	}

	// ============================================
	// Balance Query Optimizations
	// ============================================
	// Composite index for balance time-range queries (balances.go:173-191)

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_balances_account_asset_time_range
		    ON balances (account_id, asset, last_updated_at, created_at);
	`)
	if err != nil {
		return err
	}

	// ============================================
	// Connector Deletion Optimizations
	// ============================================
	// Composite indexes for CASCADE DELETE performance when deleting connectors (connectors.go:169-175)

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_payments_connector_created_sort
		    ON payments (connector_id, created_at DESC, sort_id DESC);
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_accounts_connector_created_sort
		    ON accounts (connector_id, created_at DESC, sort_id DESC);
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_balances_connector_account_asset
		    ON balances (connector_id, account_id, asset);
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_payment_initiations_connector_created_sort
		    ON payment_initiations (connector_id, created_at DESC, sort_id DESC);
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_payment_initiation_adjustments_pi_created_sort
		    ON payment_initiation_adjustments (payment_initiation_id, created_at DESC, sort_id DESC);
	`)
	if err != nil {
		return err
	}

	// ============================================
	// Partial Indexes
	// ============================================
	// Partial index for active (non-deleted) connectors

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_connectors_active_id_name
		    ON connectors (id, name, created_at)
		    WHERE scheduled_for_deletion = false;
	`)
	if err != nil {
		return err
	}

	// ============================================
	// Payment Initiation Related Optimizations
	// ============================================
	// Covering index for payment initiation related payments lookup

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_payment_initiation_related_payments_both_ids
		    ON payment_initiation_related_payments (payment_initiation_id, payment_id, created_at DESC);
	`)
	if err != nil {
		return err
	}

	// ============================================
	// Additional Composite Indexes
	// ============================================
	// Composite index for pool account lookups

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_pool_accounts_pool_account_connector
		    ON pool_accounts (pool_id, account_id, connector_id);
	`)
	if err != nil {
		return err
	}

	// Composite index for workflow instances by connector and schedule

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_workflows_instances_connector_schedule
		    ON workflows_instances (connector_id, schedule_id, created_at DESC);
	`)
	if err != nil {
		return err
	}

	// Composite index for tasks by connector and status

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_tasks_connector_status_created
		    ON tasks (connector_id, status, created_at DESC)
		    WHERE connector_id IS NOT NULL;
	`)
	if err != nil {
		return err
	}

	// ============================================
	// PSU and Open Banking Indexes
	// ============================================
	// Composite indexes for PSU and open banking connection queries

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_payments_psu_connector_obc
		    ON payments (psu_id, connector_id, open_banking_connection_id)
		    WHERE psu_id IS NOT NULL;
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_accounts_psu_connector_obc
		    ON accounts (psu_id, connector_id, open_banking_connection_id)
		    WHERE psu_id IS NOT NULL;
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY idx_balances_psu_connector_obc
		    ON balances (psu_id, connector_id, open_banking_connection_id)
		    WHERE psu_id IS NOT NULL;
	`)
	if err != nil {
		return err
	}

	return nil
}
