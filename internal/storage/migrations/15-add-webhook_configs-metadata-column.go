package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func AddWebhooksConfigsMetadata(ctx context.Context, db bun.IDB) error {
	_, err := db.ExecContext(ctx, `
		ALTER TABLE webhooks_configs ADD COLUMN IF NOT EXISTS metadata jsonb default '{}'::jsonb;
	`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		ALTER TABLE webhooks_configs ALTER COLUMN metadata SET NOT NULL;
	`)
	if err != nil {
		return err
	}
	return nil
}
