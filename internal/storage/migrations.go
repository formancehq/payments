package storage

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	"github.com/formancehq/go-libs/v2/migrations"
	paymentsMigration "github.com/formancehq/payments/internal/storage/migrations"
	"github.com/uptrace/bun"
)

// EncryptionKey is set from the migration utility to specify default encryption key to migrate to.
// This can remain empty. Then the config will be removed.
//
//nolint:gochecknoglobals // This is a global variable by design.
var EncryptionKey string

//go:embed migrations/0-init-schema.sql
var initSchema string

//go:embed migrations/5-migrate-bank-accounts-from-v2.sql
var migrateBankAccountsFromV2 string

//go:embed migrations/7-migrate-transfer-initiations-from-v2.sql
var migrateTransferInitiationsFromV2 string

//go:embed migrations/9-migrate-pools-from-v2.sql
var migratePoolsFromV2 string

func registerMigrations(migrator *migrations.Migrator, encryptionKey string) {
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
		migrations.Migration{
			Name: "migrate connectors from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					return paymentsMigration.MigrateConnectorsFromV2(ctx, db, encryptionKey)
				})
			},
		},
		migrations.Migration{
			Name: "migrate accounts events from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					return paymentsMigration.MigrateAccountEventsFromV2(ctx, db)
				})
			},
		},
		migrations.Migration{
			Name: "migrate payments adjustments events from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					return paymentsMigration.MigratePaymentsAdjustmentsFromV2(ctx, db)
				})
			},
		},
		migrations.Migration{
			Name: "migrate payments events from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					return paymentsMigration.MigratePaymentsFromV2(ctx, db)
				})
			},
		},
		migrations.Migration{
			Name: "migrate balances events from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					return paymentsMigration.MigrateBalancesFromV2(ctx, db)
				})
			},
		},
		migrations.Migration{
			Name: "migrate bank accounts from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					_, err := tx.ExecContext(ctx, migrateBankAccountsFromV2)
					return err
				})
			},
		},
		migrations.Migration{
			Name: "fix missing reference for v2 transfer initiations",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					return paymentsMigration.FixMissingReferenceTransferInitiation(ctx, db)
				})
			},
		},
		migrations.Migration{
			Name: "migrate transfer initiations from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					_, err := tx.ExecContext(ctx, migrateTransferInitiationsFromV2)
					return err
				})
			},
		},
		migrations.Migration{
			Name: "migrate payment initiation adjustments from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					return paymentsMigration.MigrateTransferInitiationAdjustmentsFromV2(ctx, db)
				})
			},
		},
		migrations.Migration{
			Name: "migrate pools from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					_, err := tx.ExecContext(ctx, migratePoolsFromV2)
					return err
				})
			},
		},
		migrations.Migration{
			Name: "migrate payment initiation reversals from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					return paymentsMigration.MigrateTransferReversalsFromV2(ctx, db)
				})
			},
		},
	)
}

func getMigrator(db *bun.DB, encryptionKey string, opts ...migrations.Option) *migrations.Migrator {
	migrator := migrations.NewMigrator(db, opts...)
	registerMigrations(migrator, encryptionKey)
	return migrator
}

func Migrate(ctx context.Context, db bun.IDB, encryptionKey string) error {
	d, ok := db.(*bun.DB)
	if !ok {
		return fmt.Errorf("db of type %T was not of expected *bun.DB type", db)
	}

	options := []migrations.Option{
		migrations.WithTableName("goose_db_version_v3"),
	}

	return getMigrator(d, encryptionKey, options...).Up(ctx)
}
