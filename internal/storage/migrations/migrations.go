package migrations

import (
	"context"
	"database/sql"
	_ "embed"

	"github.com/formancehq/go-libs/v2/logging"
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

func registerMigrations(logger logging.Logger, migrator *migrations.Migrator, encryptionKey string) {
	migrator.RegisterMigrations(
		migrations.Migration{
			Name: "init schema",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running init schema migration...")
					_, err := tx.ExecContext(ctx, initSchema)
					logger.WithField("error", err).Info("finished running init schema migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "migrate connectors from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running migrate connectors from v2 migration...")
					err := MigrateConnectorsFromV2(ctx, logger, db, encryptionKey)
					logger.WithField("error", err).Info("finished running migrate connectors from v2 migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "migrate accounts events from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running migrate accounts events from v2 migration...")
					err := MigrateAccountEventsFromV2(ctx, logger, db)
					logger.WithField("error", err).Info("finished running migrate accounts events from v2 migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "migrate payments adjustments events from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running migrate payments adjustments events from v2 migration...")
					err := MigratePaymentsAdjustmentsFromV2(ctx, logger, db)
					logger.WithField("error", err).Info("finished running migrate payments adjustments events from v2 migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "migrate payments events from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running migrate payments events from v2 migration...")
					err := MigratePaymentsFromV2(ctx, db)
					logger.WithField("error", err).Info("finished running migrate payments events from v2 migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "migrate balances events from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running migrate balances events from v2 migration...")
					err := MigrateBalancesFromV2(ctx, logger, db)
					logger.WithField("error", err).Info("finished running migrate balances events from v2 migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "migrate bank accounts from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running migrate bank accounts from v2 migration...")
					_, err := tx.ExecContext(ctx, migrateBankAccountsFromV2)
					logger.WithField("error", err).Info("finished running migrate bank accounts from v2 migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "fix missing reference for v2 transfer initiations",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running fix missing reference for v2 transfer initiations migration...")
					err := FixMissingReferenceTransferInitiation(ctx, db)
					logger.WithField("error", err).Info("finished running fix missing reference for v2 transfer initiations migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "migrate transfer initiations from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running migrate transfer initiations from v2 migration...")
					_, err := tx.ExecContext(ctx, migrateTransferInitiationsFromV2)
					logger.WithField("error", err).Info("finished running migrate transfer initiations from v2 migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "migrate payment initiation adjustments from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running migrate payment initiation adjustments from v2 migration...")
					err := MigrateTransferInitiationAdjustmentsFromV2(ctx, db)
					logger.WithField("error", err).Info("finished running migrate payment initiation adjustments from v2 migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "migrate pools from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running migrate pools from v2 migration...")
					_, err := tx.ExecContext(ctx, migratePoolsFromV2)
					logger.WithField("error", err).Info("finished running migrate pools from v2 migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "migrate payment initiation reversals from v2",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running migrate payment initiation reversals from v2 migration...")
					err := MigrateTransferReversalsFromV2(ctx, db)
					logger.WithField("error", err).Info("finished running migrate payment initiation reversals from v2 migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "add connector reference",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running add connector reference migration...")
					err := AddReferenceForConnector(ctx, db)
					logger.WithField("error", err).Info("finished add connector reference migration")
					return err
				})
			},
		},
	)
}

func GetMigrator(logger logging.Logger, db *bun.DB, encryptionKey string, opts ...migrations.Option) *migrations.Migrator {
	migrator := migrations.NewMigrator(db, opts...)
	registerMigrations(logger, migrator, encryptionKey)
	return migrator
}
