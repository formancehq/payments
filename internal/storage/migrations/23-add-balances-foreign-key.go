package migrations

import (
	"context"
	"github.com/uptrace/bun"
)

func AddBalancesForeignKey(ctx context.Context, db bun.IDB) error {
	// Add the new columns psuId & openBankingConnectionId
	_, err := db.ExecContext(ctx, `
		ALTER TABLE balances 
		    ADD COLUMN IF NOT EXISTS psu_id uuid;
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		ALTER TABLE balances 
		    ADD COLUMN IF NOT EXISTS open_banking_connection_id varchar;
	`)
	if err != nil {
		return err
	}

	// Create the indices
	_, err = db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS balances_psu_id_idx 
			ON balances (psu_id);
	`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS balances_open_banking_connection_id_idx 
			ON balances (psu_id, connector_id, open_banking_connection_id);
	`)
	if err != nil {
		return err
	}

	// Add constraints.
	// We set NOT VALID to not lock the table.
	_, err = db.ExecContext(ctx, `
		 ALTER TABLE balances
			ADD CONSTRAINT balances_psu_id_fk FOREIGN KEY (psu_id) 
			REFERENCES payment_service_users (id) 
			ON DELETE CASCADE 
			NOT VALID;
	`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		ALTER TABLE balances
			ADD CONSTRAINT balances_open_banking_connection_id_fk FOREIGN KEY (psu_id, connector_id, open_banking_connection_id)
			REFERENCES open_banking_connections (psu_id, connector_id, connection_id)
			ON DELETE CASCADE 
			NOT VALID;
`)
	if err != nil {
		return err
	}

	// Validate the constraints (Segscan, but without exclusive lock, concurrent sessions can R/W)
	_, err = db.ExecContext(ctx, `
		ALTER TABLE balances
			VALIDATE CONSTRAINT balances_psu_id_fk;
`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		ALTER TABLE balances
			VALIDATE CONSTRAINT balances_open_banking_connection_id_fk;
`)
	if err != nil {
		return err
	}

	return nil
}
