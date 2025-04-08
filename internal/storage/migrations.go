package storage

import (
	"context"
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/migrations"
	paymentsMigration "github.com/formancehq/payments/internal/storage/migrations"
	"github.com/uptrace/bun"
)

// EncryptionKey is set from the migration utility to specify default encryption key to migrate to.
// This can remain empty. Then the config will be removed.
//
//nolint:gochecknoglobals // This is a global variable by design.
var EncryptionKey string

func Migrate(ctx context.Context, logger logging.Logger, db bun.IDB, encryptionKey string) error {
	d, ok := db.(*bun.DB)
	if !ok {
		return fmt.Errorf("db of type %T was not of expected *bun.DB type", db)
	}

	options := []migrations.Option{
		migrations.WithTableName("goose_db_version_v3"),
	}

	return paymentsMigration.GetMigrator(logger, d, encryptionKey, options...).Up(ctx)
}
