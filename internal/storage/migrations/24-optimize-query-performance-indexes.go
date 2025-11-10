package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

// OptimizeQueryPerformanceIndexes adds comprehensive indexes to improve query performance across the application.
// This migration focuses on:
// 1. JSONB metadata queries (GIN indexes)
// 2. Payment and adjustment queries with LATERAL JOINs
// 3. Connector deletion performance (cascade optimization)
// 4. Reference lookups
// 5. Balance time-series queries
// 6. Partial indexes for filtered queries
func OptimizeQueryPerformanceIndexes(ctx context.Context, db bun.IDB) error {
	queries := []struct {
		name  string
		query string
	}{
		// JSONB GIN Indexes - dramatically improve metadata filtering performance
		{
			name: "payments_metadata_gin",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS payments_metadata_gin
				ON payments USING gin (metadata)`,
		},
		{
			name: "accounts_metadata_gin",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS accounts_metadata_gin
				ON accounts USING gin (metadata)`,
		},
		{
			name: "payment_adjustments_metadata_gin",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_adjustments_metadata_gin
				ON payment_adjustments USING gin (metadata)`,
		},
		{
			name: "payment_initiations_metadata_gin",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiations_metadata_gin
				ON payment_initiations USING gin (metadata)`,
		},
		{
			name: "payment_initiation_adjustments_metadata_gin",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiation_adjustments_metadata_gin
				ON payment_initiation_adjustments USING gin (metadata)`,
		},
		{
			name: "payment_initiation_reversals_metadata_gin",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiation_reversals_metadata_gin
				ON payment_initiation_reversals USING gin (metadata)`,
		},
		{
			name: "bank_accounts_metadata_gin",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS bank_accounts_metadata_gin
				ON bank_accounts USING gin (metadata)`,
		},

		// Composite index for LATERAL JOIN in PaymentsList (payments.go:432-439)
		{
			name: "payment_adjustments_payment_created_sort",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_adjustments_payment_created_sort
				ON payment_adjustments (payment_id, created_at DESC, sort_id DESC)`,
		},

		// Composite index for PaymentsGetByReference (payments.go:254-264)
		{
			name: "payments_connector_reference",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS payments_connector_reference
				ON payments (connector_id, reference)`,
		},

		// Composite index for balance time-range queries (balances.go:173-191)
		{
			name: "balances_account_asset_time_range",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS balances_account_asset_time_range
				ON balances (account_id, asset, last_updated_at, created_at)`,
		},

		// Composite indexes for CASCADE DELETE performance when deleting connectors
		{
			name: "payments_connector_created_sort",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS payments_connector_created_sort
				ON payments (connector_id, created_at DESC, sort_id DESC)`,
		},
		{
			name: "accounts_connector_created_sort",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS accounts_connector_created_sort
				ON accounts (connector_id, created_at DESC, sort_id DESC)`,
		},
		{
			name: "balances_connector_account_asset",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS balances_connector_account_asset
				ON balances (connector_id, account_id, asset)`,
		},
		{
			name: "payment_initiations_connector_created_sort",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiations_connector_created_sort
				ON payment_initiations (connector_id, created_at DESC, sort_id DESC)`,
		},
		{
			name: "payment_initiation_adjustments_pi_created_sort",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiation_adjustments_pi_created_sort
				ON payment_initiation_adjustments (payment_initiation_id, created_at DESC, sort_id DESC)`,
		},

		// Partial index for active connectors
		{
			name: "connectors_active_id_name",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS connectors_active_id_name
				ON connectors (id, name, created_at)
				WHERE scheduled_for_deletion = false`,
		},

		// Covering index for payment initiation related payments lookup
		{
			name: "payment_initiation_related_payments_both_ids",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiation_related_payments_both_ids
				ON payment_initiation_related_payments (payment_initiation_id, payment_id, created_at DESC)`,
		},

		// Composite index for pool account lookups
		{
			name: "pool_accounts_pool_account_connector",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS pool_accounts_pool_account_connector
				ON pool_accounts (pool_id, account_id, connector_id)`,
		},

		// Composite index for workflow instances by connector and schedule
		{
			name: "workflows_instances_connector_schedule",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS workflows_instances_connector_schedule
				ON workflows_instances (connector_id, schedule_id, created_at DESC)`,
		},

		// Composite index for tasks by connector and status
		{
			name: "tasks_connector_status_created",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS tasks_connector_status_created
				ON tasks (connector_id, status, created_at DESC)
				WHERE connector_id IS NOT NULL`,
		},

		// Composite indexes for PSU and open banking connection queries
		{
			name: "payments_psu_connector_obc",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS payments_psu_connector_obc
				ON payments (psu_id, connector_id, open_banking_connection_id)
				WHERE psu_id IS NOT NULL`,
		},
		{
			name: "accounts_psu_connector_obc",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS accounts_psu_connector_obc
				ON accounts (psu_id, connector_id, open_banking_connection_id)
				WHERE psu_id IS NOT NULL`,
		},
		{
			name: "balances_psu_connector_obc",
			query: `CREATE INDEX CONCURRENTLY IF NOT EXISTS balances_psu_connector_obc
				ON balances (psu_id, connector_id, open_banking_connection_id)
				WHERE psu_id IS NOT NULL`,
		},
	}

	// Execute each index creation separately to ensure better error handling
	// and to allow partial success (some indexes might already exist in production)
	for _, q := range queries {
		_, err := db.ExecContext(ctx, q.query)
		if err != nil {
			return fmt.Errorf("failed to create index %s: %w", q.name, err)
		}
	}

	return nil
}
