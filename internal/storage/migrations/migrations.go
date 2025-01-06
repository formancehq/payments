package migrations

import (
	"context"
	"database/sql"
	_ "embed"

	"github.com/formancehq/go-libs/v2/migrations"
	"github.com/uptrace/bun"
)

//go:embed 0-init-schema.sql
var initSchema string

//go:embed 5-migrate-bank-accounts-from-v2.sql
var migrateBankAccountsFromV2 string

//go:embed 7-migrate-transfer-initiations-from-v2.sql
var migrateTransferInitiationsFromV2 string

//go:embed 9-migrate-pools-from-v2.sql
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
					return MigrateConnectorsFromV2(ctx, db, encryptionKey)
				})
			},
		},
		migrations.Migration{
			Name: "migrate accounts events from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					return MigrateAccountEventsFromV2(ctx, db)
				})
			},
		},
		migrations.Migration{
			Name: "migrate payments adjustments events from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					return MigratePaymentsAdjustmentsFromV2(ctx, db)
				})
			},
		},
		migrations.Migration{
			Name: "migrate payments events from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					return MigratePaymentsFromV2(ctx, db)
				})
			},
		},
		migrations.Migration{
			Name: "migrate balances events from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					return MigrateBalancesFromV2(ctx, db)
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
					return FixMissingReferenceTransferInitiation(ctx, db)
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
					return MigrateTransferInitiationAdjustmentsFromV2(ctx, db)
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
					return MigrateTransferReversalsFromV2(ctx, db)
				})
			},
		},
	)
}

func GetMigrator(db *bun.DB, encryptionKey string, opts ...migrations.Option) *migrations.Migrator {
	migrator := migrations.NewMigrator(db, opts...)
	registerMigrations(migrator, encryptionKey)
	return migrator
}
