package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

// AddPSUConnectionPaymentsAccountsAsync mirrors 23-psu-connection-payments-accounts-async.sql
// Each statement is executed in its own autocommitted transaction (no surrounding tx),
// similar to migration 17-add-foreign-key-indices.
func AddPSUConnectionPaymentsAccountsAsync(ctx context.Context, db bun.IDB) error {
	stmts := []string{
		`ALTER TABLE IF EXISTS payments ADD COLUMN IF NOT EXISTS psu_id uuid;`,
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS payments_psu_id_idx ON payments (psu_id);`,

		`DO $$
BEGIN
	PERFORM 1 FROM pg_catalog.pg_constraint c
	WHERE c.conname = 'payments_psu_id_fk'
	  AND c.conrelid = 'public.payments'::regclass
	  AND c.contype = 'f';
	IF NOT FOUND THEN
		ALTER TABLE public.payments
			ADD CONSTRAINT payments_psu_id_fk
			FOREIGN KEY (psu_id)
			REFERENCES payment_service_users (id)
			ON DELETE CASCADE
			NOT VALID;
	END IF;
END$$;`,

		`ALTER TABLE IF EXISTS payments ADD COLUMN IF NOT EXISTS open_banking_connection_id varchar;`,
		`DO $$
BEGIN
	PERFORM 1 FROM pg_catalog.pg_constraint c
	WHERE c.conname = 'payments_open_banking_connection_id_fk'
	  AND c.conrelid = 'public.payments'::regclass
	  AND c.contype = 'f';
	IF NOT FOUND THEN
		ALTER TABLE public.payments
			ADD CONSTRAINT payments_open_banking_connection_id_fk
			FOREIGN KEY (psu_id, connector_id, open_banking_connection_id)
			REFERENCES open_banking_connections (psu_id, connector_id, connection_id)
			ON DELETE CASCADE
			NOT VALID;
	END IF;
END$$;`,
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS payments_psu_conn_obc_idx ON payments (psu_id, connector_id, open_banking_connection_id);`,

		`ALTER TABLE IF EXISTS accounts ADD COLUMN IF NOT EXISTS psu_id uuid;`,
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS accounts_psu_id_idx ON accounts (psu_id);`,

		`DO $$
BEGIN
	PERFORM 1 FROM pg_catalog.pg_constraint c
	WHERE c.conname = 'accounts_psu_id_fk'
	  AND c.conrelid = 'public.accounts'::regclass
	  AND c.contype = 'f';
	IF NOT FOUND THEN
		ALTER TABLE public.accounts
			ADD CONSTRAINT accounts_psu_id_fk
			FOREIGN KEY (psu_id)
			REFERENCES payment_service_users (id)
			ON DELETE CASCADE
			NOT VALID;
	END IF;
END$$;`,

		`ALTER TABLE IF EXISTS accounts ADD COLUMN IF NOT EXISTS open_banking_connection_id varchar;`,
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS accounts_psu_conn_obc_idx ON accounts (psu_id, connector_id, open_banking_connection_id);`,
		`DO $$
BEGIN
	PERFORM 1 FROM pg_catalog.pg_constraint c
	WHERE c.conname = 'accounts_open_banking_connection_id_fk'
	  AND c.conrelid = 'public.accounts'::regclass
	  AND c.contype = 'f';
	IF NOT FOUND THEN
		ALTER TABLE public.accounts
			ADD CONSTRAINT accounts_open_banking_connection_id_fk
			FOREIGN KEY (psu_id, connector_id, open_banking_connection_id)
			REFERENCES open_banking_connections (psu_id, connector_id, connection_id)
			ON DELETE CASCADE
			NOT VALID;
	END IF;
END$$;`,

		`ALTER TABLE public.payments VALIDATE CONSTRAINT payments_open_banking_connection_id_fk`,
		`ALTER TABLE public.payments VALIDATE CONSTRAINT payments_psu_id_fk`,
		`ALTER TABLE public.accounts VALIDATE CONSTRAINT accounts_psu_id_fk`,
		`ALTER TABLE public.accounts VALIDATE CONSTRAINT accounts_open_banking_connection_id_fk`,
	}

	for i, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration 23 statement %d failed: %w", i+1, err)
		}
	}
	return nil
}
