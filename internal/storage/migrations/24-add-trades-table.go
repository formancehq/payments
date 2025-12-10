package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func AddTradesTable(ctx context.Context, db bun.IDB) error {
	// Create trades table
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS trades (
			-- Auto-increment fields
			sort_id bigserial not null,

			-- Mandatory fields
			id varchar not null,
			connector_id varchar not null,
			reference text not null,
			created_at timestamp without time zone not null,
			updated_at timestamp without time zone not null,
			instrument_type text not null,
			execution_model text not null,
			market_symbol text not null,
			market_base_asset text not null,
			market_quote_asset text not null,
			side text not null,
			status text not null,
			requested jsonb not null default '{}'::jsonb,
			executed jsonb not null,
			fills jsonb not null default '[]'::jsonb,
			legs jsonb not null default '[]'::jsonb,
			raw json not null,

			-- Optional fields
			portfolio_account_id varchar,
			order_type text,
			time_in_force text,

			-- Optional with defaults
			fees jsonb not null default '[]'::jsonb,
			metadata jsonb not null default '{}'::jsonb,

			-- Primary key
			primary key (id)
		);
	`)
	if err != nil {
		return err
	}

	// Create indices
	_, err = db.ExecContext(ctx, `
		CREATE INDEX trades_created_at_sort_id ON trades (created_at, sort_id);
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX trades_connector_id ON trades (connector_id);
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX trades_portfolio_account_id ON trades (portfolio_account_id);
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX trades_status ON trades (status);
	`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE INDEX trades_market_symbol ON trades (market_symbol);
	`)
	if err != nil {
		return err
	}

	// Add foreign key constraint
	_, err = db.ExecContext(ctx, `
		ALTER TABLE trades 
			ADD CONSTRAINT trades_connector_id_fk 
			FOREIGN KEY (connector_id) 
			REFERENCES connectors (id) 
			ON DELETE CASCADE;
	`)
	if err != nil {
		return err
	}

	return nil
}

