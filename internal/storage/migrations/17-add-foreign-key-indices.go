package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

// AddForeignKeyIndices is adding a lot of missing indices; note that some indices created here are not mapping 1:1 to "real" foreign key constrains
func AddForeignKeyIndices(ctx context.Context, db bun.IDB) error {
	// While we don't have a FK from Balance to Account, as we have some balances being created before the account,
	// we do create the related index.
	// TODO -- Do we need to add FK on connector? It's sometimes missing
	// TODO -- BankAccountRelatedAccount -> Account
	// TODO -- PoolAccount -> Connector, Account
	// TODO -- payment_initiation_related_payments -> payments
	// TODO -- payment -> accounts (source,dest)
	// TODO -- payment_initiation -> accounts(source,dest)

	// Accounts
	_, err := db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS accounts_connector_id ON accounts (connector_id);
	`)
	if err != nil {
		return err
	}

	// Balances
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS balances_account_id ON balances (account_id);
	`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS balances_connector_id ON balances (connector_id);
	`)
	if err != nil {
		return err
	}

	// Bank_accounts
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS bank_accounts_psu_id ON bank_accounts (psu_id);
	`)
	if err != nil {
		return err
	}

	// Bank_accounts_related_accounts
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS bank_accounts_related_accounts_bank_account_id ON bank_accounts_related_accounts (bank_account_id);
	`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS bank_accounts_related_accounts_account_id ON bank_accounts_related_accounts (account_id);
	`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS bank_accounts_related_accounts_connector_id ON bank_accounts_related_accounts (connector_id);
	`)
	if err != nil {
		return err
	}

	// Connector_tasks_tree
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS connector_tasks_tree_connector_id ON connector_tasks_tree (connector_id);
	`)
	if err != nil {
		return err
	}

	// Events_sent
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS events_sent_connector_id ON events_sent (connector_id);
	`)
	if err != nil {
		return err
	}

	// Payment_adjustments
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_adjustments_payment_id ON payment_adjustments (payment_id);
	`)
	if err != nil {
		return err
	}

	// Payment_initiation_adjustments
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiation_adjustments_payment_initiation_id ON payment_initiation_adjustments (payment_initiation_id);
	`)
	if err != nil {
		return err
	}

	// Payment_initiation_related_payments
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiation_related_payments_payment_initiation_id ON payment_initiation_related_payments (payment_initiation_id);
	`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiation_related_payments_payment_id ON payment_initiation_related_payments (payment_id);
	`)
	if err != nil {
		return err
	}

	// Payment_initiation_reversal_adjustments
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiation_reversal_adjustments_pir_id
    ON payment_initiation_reversal_adjustments (payment_initiation_reversal_id);
	`)
	if err != nil {
		return err
	}

	// Payment_initiation_reversals
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiation_reversals_connector_id ON payment_initiation_reversals (connector_id);
	`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiation_reversals_payment_initiation_id ON payment_initiation_reversals (payment_initiation_id);
	`)
	if err != nil {
		return err
	}

	// Payment_initiations
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiations_connector_id ON payment_initiations (connector_id);
	`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiations_source_account_id ON payment_initiations (source_account_id);
	`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS payment_initiations_destination_account_id ON payment_initiations (destination_account_id);
	`)
	if err != nil {
		return err
	}

	// Payments
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS payments_connector_id ON payments (connector_id);
	`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS payments_source_account_id ON payments (source_account_id);
	`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS payments_destination_account_id ON payments (destination_account_id);
	`)
	if err != nil {
		return err
	}

	// Pool_accounts
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS pool_accounts_pool_id ON pool_accounts (pool_id);
	`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS pool_accounts_account_id ON pool_accounts (account_id);
	`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS pool_accounts_connector_id ON pool_accounts (connector_id);
	`)
	if err != nil {
		return err
	}

	// Schedules
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS schedules_connector_id ON schedules (connector_id);
	`)
	if err != nil {
		return err
	}

	// States
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS states_connector_id ON states (connector_id);
	`)
	if err != nil {
		return err
	}

	// Webhooks
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS webhooks_connector_id ON webhooks (connector_id);
	`)
	if err != nil {
		return err
	}

	// Workflows_instances
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS workflows_instances_schedule_id ON workflows_instances (schedule_id);
	`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		CREATE INDEX CONCURRENTLY IF NOT EXISTS workflows_instances_connector_id ON workflows_instances (connector_id);
	`)
	if err != nil {
		return err
	}

	return nil
}
