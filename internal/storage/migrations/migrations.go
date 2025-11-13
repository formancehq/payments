package migrations

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/migrations"
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

//go:embed 13-connector-providers-lowercase.sql
var connectorProvidersLower string

//go:embed 14-create-psu-tables.sql
var psuTableCreation string

//go:embed 16-webhooks-idempotency-key.sql
var webhooksIdempotencyKey string

//go:embed 17-add-language-psu.sql
var psuLanguageColumn string

//go:embed 18-bank-bridges-connections.sql
var bankBridgesConnections string

//go:embed 19-bank-bridge-psp-user-id.sql
var bankBridgePSPUserId string

//go:embed 20-psu-bank-bridges-connection-updated-at.sql
var psuBankBridgeConnectionUpdatedAt string

// 21 is not used anymore, the file is kept for historical reference (some staging env have run it)

//go:embed 22-rename-bank-bridges-open-banking.sql
var renameBankBridgesOpenBanking string

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
		migrations.Migration{
			Name: "add payment_initiation_adjustments indexes",
			Up: func(ctx context.Context, db bun.IDB) error {
				logger.Info("running add payment_initiation_adjustments index migration...")
				err := AddPaymentInitiationAdjustmentsIndexes(ctx, db)
				logger.WithField("error", err).Info("finished add payment_initiation_adjustments index migration")
				return err
			},
		},
		migrations.Migration{
			Name: "connector providers lowercase",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running connector providers lowercase migration...")
					_, err := tx.ExecContext(ctx, connectorProvidersLower)
					logger.WithField("error", err).Info("finished connector providers lowercase migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "create psu tables",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running psu table creations migration...")
					_, err := tx.ExecContext(ctx, psuTableCreation)
					logger.WithField("error", err).Info("finished running psu table creations migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "add webhooks_configs metadata column",
			Up: func(ctx context.Context, db bun.IDB) error {
				logger.Info("running webhooks_configs metadata addition migration...")
				err := AddWebhooksConfigsMetadata(ctx, db)
				logger.WithField("error", err).Info("finished running webhooks_configs metadata addition migration")
				return err
			},
		},
		migrations.Migration{
			Name: "webhooks idempotency key",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running webhooks idempotency key migration...")
					_, err := tx.ExecContext(ctx, webhooksIdempotencyKey)
					logger.WithField("error", err).Info("finished running webhooks idempotency key migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "foreign key indices creation",
			Up: func(ctx context.Context, db bun.IDB) error {
				logger.Info("running foreign key indices creation migration...")
				err := AddForeignKeyIndices(ctx, db)
				logger.WithField("error", err).Info("finished running foreign key indices creation migration")
				return err
			},
		},
		migrations.Migration{
			Name: "add language to psu",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running add language to psu migration...")
					_, err := tx.ExecContext(ctx, psuLanguageColumn)
					logger.WithField("error", err).Info("finished running add language to psu migration")
					return err
				})
			},
		},

		// Bank bridges was the former name of OpenBanking
		migrations.Migration{
			Name: "bank bridges connections",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running add bank bridges connections migration...")
					_, err := tx.ExecContext(ctx, bankBridgesConnections)
					logger.WithField("error", err).Info("finished running add bank bridges connections migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "bank bridge psp user id",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running bank bridge psp user id migration...")
					_, err := tx.ExecContext(ctx, bankBridgePSPUserId)
					logger.WithField("error", err).Info("finished running bank bridge psp user id migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "psu bank bridge connection updated at",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running psu bank bridge connection updated at migration...")
					_, err := tx.ExecContext(ctx, psuBankBridgeConnectionUpdatedAt)
					logger.WithField("error", err).Info("finished running psu bank bridge connection updated at migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "psu connection payments accounts",
			Up: func(ctx context.Context, db bun.IDB) error {
				// Migration 21 ran in some environments, but not others -- we keep the numbering but we
				// skip the actual migration here. Migration 23 is the replacement (we're removing the table locking on
				// payment, accounts etc)
				//return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
				//	logger.Info("running psu connection payments accounts migration...")
				//	_, err := tx.ExecContext(ctx, psuConnectionPaymentsAccounts)
				//	logger.WithField("error", err).Info("finished running psu connection payments accounts migration")
				//	return err
				//})
				return nil
			},
		},
		migrations.Migration{
			Name: "rename bank bridges to open banking",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running rename bank bridges to open banking migration...")
					_, err := tx.ExecContext(ctx, renameBankBridgesOpenBanking)
					logger.WithField("error", err).Info("finished running rename bank bridges open banking migration")
					return err
				})
			},
		},
		migrations.Migration{
			Name: "psu connection payments accounts async",
			Up: func(ctx context.Context, db bun.IDB) error {
				logger.Info("running psu connection payments accounts async migration...")
				// Guard: IDB must be *bun.DB, not *bun.Tx.
				if _, ok := db.(*bun.Tx); ok {
					return fmt.Errorf("migration 23 must not run inside a transaction; pass a *bun.DB")
				}
				err := AddPSUConnectionPaymentsAccountsAsync(ctx, db)
				logger.WithField("error", err).Info("finished running psu connection payments accounts async migration")
				return err
			},
		},
		migrations.Migration{
			Name: "add connection and psu foreign keys on balances",
			Up: func(ctx context.Context, db bun.IDB) error {
				logger.Info("running add balances foreign key migration...")
				err := AddBalancesForeignKey(ctx, db)
				logger.WithField("error", err).Info("finished running add balances foreign key migration")
				return err
			},
		},
		migrations.Migration{
			Name: "add trades table",
			Up: func(ctx context.Context, db bun.IDB) error {
				return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
					logger.Info("running add trades table migration...")
					err := AddTradesTable(ctx, tx)
					logger.WithField("error", err).Info("finished running add trades table migration")
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
