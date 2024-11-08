package storage

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	"github.com/formancehq/go-libs/v2/migrations"
	"github.com/uptrace/bun"
)

// EncryptionKey is set from the migration utility to specify default encryption key to migrate to.
// This can remain empty. Then the config will be removed.
//
//nolint:gochecknoglobals // This is a global variable by design.
var EncryptionKey string

//go:embed migrations/0-init-schema.sql
var initSchema string

func registerMigrations(migrator *migrations.Migrator) {
	migrator.RegisterMigrations(
		migrations.Migration{
			Name: "init schema",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					_, err := tx.ExecContext(ctx, initSchema)
					return err
				})
			},
		},
	)
}

func getMigrator(db *bun.DB, opts ...migrations.Option) *migrations.Migrator {
	migrator := migrations.NewMigrator(db, opts...)
	registerMigrations(migrator)
	return migrator
}

func Migrate(ctx context.Context, db bun.IDB) error {
	d, ok := db.(*bun.DB)
	if !ok {
		return fmt.Errorf("db of type %T was not of expected *bun.DB type")
	}
	return getMigrator(d).Up(ctx)
}
