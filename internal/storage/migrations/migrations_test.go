package migrations

import (
	"context"
	_ "embed"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunconnect"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/migrations"
	testmigrations "github.com/formancehq/go-libs/v3/testing/migrations"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/extra/bundebug"
)

var (
	//go:embed v2_migrations_test.sql
	v2TestSQL string

	testConnectorID = models.MustConnectorIDFromString("eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9")
)

func fillDBTestMigrations(t *testing.T, db *bun.DB) {
	_, err := db.Exec(v2TestSQL)
	require.NoError(t, err)
}

func TestMigrationsWithoutV2(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	pgDatabase := srv.NewDatabase(t)
	db, err := bunconnect.OpenSQLDB(ctx, pgDatabase.ConnectionOptions())
	require.NoError(t, err)

	if testing.Verbose() {
		db.AddQueryHook(bundebug.NewQueryHook())
	}

	migrator := GetMigrator(logging.Testing(), db, "default-encryption-key", []migrations.Option{
		migrations.WithTableName("goose_db_version_v3"),
	}...)
	test := testmigrations.NewMigrationTest(t, migrator, db)
	test.Run()
}

func TestMigrationsWithV2(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	pgDatabase := srv.NewDatabase(t)
	db, err := bunconnect.OpenSQLDB(ctx, pgDatabase.ConnectionOptions())
	require.NoError(t, err)

	if testing.Verbose() {
		db.AddQueryHook(bundebug.NewQueryHook())
	}

	// Add v2 schema and data
	fillDBTestMigrations(t, db)

	migrator := GetMigrator(logging.Testing(), db, "default-encryption-key", []migrations.Option{
		migrations.WithTableName("goose_db_version_v3"),
	}...)
	test := testmigrations.NewMigrationTest(t, migrator, db)
	test.Append(1, testConnectorsMigration())
	test.Append(2, testAccountsEventsMigration())
	// We're skipping 3 since 3 and 4 are related to payment events migrations
	test.Append(4, testPaymentsEventsMigration())
	test.Append(5, testBalancesEventsMigration())
	test.Append(6, testBankAccountsMigration())
	// Migrations 7, 8 and 9 are related to transfer initiations
	test.Append(9, testTransferInitiationsMigrations())
	test.Append(10, testPoolsMigrations())
	test.Append(11, testPaymentReversalsMigrations())
	test.Append(12, testReferenceConnectorMigrations())
	test.Append(14, testConnectorsProviderLowercaseMigration())
	test.Run()
}

func testConnectorsMigration() testmigrations.Hook {
	return testmigrations.Hook{
		After: func(ctx context.Context, t *testing.T, db bun.IDB) {
			count, err := db.NewSelect().TableExpr("connectors").Count(ctx)
			require.NoError(t, err)
			// We should have 10 connectors in the new v3 table and not 11
			// since the dummy pay one should have been skipped
			require.Equal(t, 10, count)

			type connector struct {
				connectorID string
				provider    string
			}

			for _, c := range []connector{
				{"eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9", "moneycorp"},
				{"eyJQcm92aWRlciI6ImFkeWVuIiwiUmVmZXJlbmNlIjoiNGEwNzUyYWUtYWIxYS00NWI4LTgyNzItMjI2ODA3OTE2NTQ0In0", "adyen"},
				{"eyJQcm92aWRlciI6ImF0bGFyIiwiUmVmZXJlbmNlIjoiN2JkZTk4NGUtYzY1OC00MzNiLWE1OGEtZTUxMGMwMTYwMDYwIn0", "atlar"},
				{"eyJQcm92aWRlciI6ImJhbmtpbmdjaXJjbGUiLCJSZWZlcmVuY2UiOiIwODQ4OWFlNC0zOGUxLTQwMTEtYjViMS1mZjkxMTliYWEzNDkifQ", "bankingcircle"},
				{"eyJQcm92aWRlciI6ImN1cnJlbmN5Y2xvdWQiLCJSZWZlcmVuY2UiOiJlNmI4OGFlZS05OTI0LTQ4ZmYtYTZkMS1mYmIwZjJjMjRkYWYifQ", "currencycloud"},
				{"eyJQcm92aWRlciI6ImdlbmVyaWMiLCJSZWZlcmVuY2UiOiIwYmE0MDNiYi0zYzlmLTQ2OTUtYmQxZC0yYmQ5ZDdiMjgwOTQifQ", "generic"},
				{"eyJQcm92aWRlciI6Im1hbmdvcGF5IiwiUmVmZXJlbmNlIjoiZTQ0MGIyMzgtM2RkNi00YzhlLTk5MDktZTJjOTgzODA2MTgyIn0", "mangopay"},
				{"eyJQcm92aWRlciI6Im1vZHVsciIsIlJlZmVyZW5jZSI6IjYzZTZlNDIyLWQ5MWMtNDQ3YS1hODU0LTE5ODJkYTU1YzljYyJ9", "modulr"},
				{"eyJQcm92aWRlciI6InN0cmlwZSIsIlJlZmVyZW5jZSI6ImIwYzZjNTdhLTM3MDYtNDRmMi1iMDdmLTE3YjNiYTdhZDhkYyJ9", "stripe"},
				{"eyJQcm92aWRlciI6Indpc2UiLCJSZWZlcmVuY2UiOiI4OWJlZDg1MS1kMjIyLTQ2NzItYjEwYy00ZDczZWE2ZGY0NGEifQ", "wise"},
			} {
				var connectorID, provider string
				err := db.NewSelect().
					TableExpr("connectors").
					Column("id", "provider").
					Where("id = ?", c.connectorID).
					Scan(ctx, &connectorID, &provider)
				require.NoError(t, err)
				require.Equal(t, c.connectorID, connectorID)
				require.Equal(t, c.provider, provider)
			}
		},
	}
}

func testConnectorsProviderLowercaseMigration() testmigrations.Hook {
	return testmigrations.Hook{
		After: func(ctx context.Context, t *testing.T, db bun.IDB) {
			for _, provider := range []string{
				"BANKING-CIRCLE",
				"CURRENCY-CLOUD",
				"DUMMY-PAY",
				"MODULR",
				"STRIPE",
				"WISE",
				"MANGOPAY",
				"MONEYCORP",
				"ATLAR",
				"ADYEN",
				"GENERIC",
			} {
				exists, err := db.NewSelect().TableExpr("connectors").
					Where("provider = ?", provider).
					Exists(ctx)
				require.NoError(t, err)
				require.False(t, exists)
			}

			for _, provider := range []string{
				"bankingcircle",
				"currencycloud",
				"modulr",
				"stripe",
				"wise",
				"mangopay",
				"moneycorp",
				"atlar",
				"adyen",
				"generic",
			} {
				exists, err := db.NewSelect().TableExpr("connectors").
					Where("provider = ?", provider).
					Exists(ctx)
				require.NoError(t, err)
				require.True(t, exists)
			}
		},
	}
}

func testEventExists(ctx context.Context, db bun.IDB, eventID models.EventID) (bool, error) {
	return db.NewSelect().TableExpr("events_sent").Where("id = ?", eventID).Exists(ctx)
}

func testAccountsEventsMigration() testmigrations.Hook {
	accountID := models.MustAccountIDFromString("eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0")

	return testmigrations.Hook{
		After: func(ctx context.Context, t *testing.T, db bun.IDB) {
			exists, err := testEventExists(
				ctx,
				db,
				models.EventID{
					// Must be the id of the account inserted in the v2 db
					EventIdempotencyKey: models.IdempotencyKey(accountID),
					ConnectorID:         &testConnectorID,
				},
			)
			require.NoError(t, err)
			require.True(t, exists)
		},
	}
}

func testPaymentsEventsMigration() testmigrations.Hook {
	return testmigrations.Hook{
		After: func(ctx context.Context, t *testing.T, db bun.IDB) {
			paymentID := models.MustPaymentIDFromString("eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0IiwiVHlwZSI6IlBBWS1JTiJ9")
			createdAt, err := time.Parse(time.RFC3339Nano, "2025-01-07T10:24:02.854346Z")
			require.NoError(t, err)
			createdAt2, err := time.Parse(time.RFC3339Nano, "2025-01-07T11:25:02.854346Z")
			require.NoError(t, err)

			exists, err := testEventExists(
				ctx,
				db,
				models.EventID{
					EventIdempotencyKey: models.IdempotencyKey(models.PaymentAdjustmentID{
						PaymentID: *paymentID,
						Reference: "test",
						CreatedAt: createdAt.UTC(),
						Status:    models.PAYMENT_STATUS_SUCCEEDED,
					}),
					ConnectorID: &testConnectorID,
				},
			)
			require.NoError(t, err)
			require.True(t, exists)

			exists, err = testEventExists(
				ctx,
				db,
				models.EventID{
					EventIdempotencyKey: models.IdempotencyKey(models.PaymentAdjustmentID{
						PaymentID: *paymentID,
						Reference: "test2",
						CreatedAt: createdAt2.UTC(),
						Status:    models.PAYMENT_STATUS_FAILED,
					}),
					ConnectorID: &testConnectorID,
				},
			)
			require.NoError(t, err)
			require.True(t, exists)
		},
	}
}

func testBalancesEventsMigration() testmigrations.Hook {
	return testmigrations.Hook{
		After: func(ctx context.Context, t *testing.T, db bun.IDB) {
			accountID := models.MustAccountIDFromString("eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0")
			createdAt, err := time.Parse(time.RFC3339Nano, "2025-01-06T10:30:02.854346Z")
			require.NoError(t, err)
			lastUpdatedAt, err := time.Parse(time.RFC3339Nano, "2025-01-06T11:30:02.854346Z")
			require.NoError(t, err)
			balance := models.Balance{
				AccountID:     accountID,
				CreatedAt:     createdAt.UTC(),
				LastUpdatedAt: lastUpdatedAt.UTC(),
				Asset:         "USD/2",
				Balance:       big.NewInt(1000),
			}
			exists, err := testEventExists(
				ctx,
				db,
				models.EventID{
					// Must be the id of the account inserted in the v2 db
					EventIdempotencyKey: balance.IdempotencyKey(),
					ConnectorID:         &testConnectorID,
				},
			)
			require.NoError(t, err)
			require.True(t, exists)
		},
	}
}

func testBankAccountsMigration() testmigrations.Hook {
	return testmigrations.Hook{
		After: func(ctx context.Context, t *testing.T, db bun.IDB) {
			count, err := db.NewSelect().TableExpr("bank_accounts").Count(ctx)
			require.NoError(t, err)
			require.Equal(t, 1, count)

			exists, err := db.NewSelect().TableExpr("bank_accounts").
				Where("id = ?", "83064af3-bb81-4514-a6d4-afba340825cd").
				Exists(ctx)
			require.NoError(t, err)
			require.True(t, exists)

			exists, err = db.NewSelect().TableExpr("bank_accounts_related_accounts").
				Where("bank_account_id = ?", "83064af3-bb81-4514-a6d4-afba340825cd").
				Where("account_id = ?", "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0").
				Exists(ctx)
			require.NoError(t, err)
			require.True(t, exists)
		},
	}
}

func testTransferInitiationsMigrations() testmigrations.Hook {
	return testmigrations.Hook{
		After: func(ctx context.Context, t *testing.T, db bun.IDB) {
			count, err := db.NewSelect().TableExpr("payment_initiations").Count(ctx)
			require.NoError(t, err)
			require.Equal(t, 1, count)

			exists, err := db.NewSelect().TableExpr("payment_initiations").
				Where("id = ?", "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0").
				Where("type = ?", models.PAYMENT_INITIATION_TYPE_PAYOUT).
				Exists(ctx)
			require.NoError(t, err)
			require.True(t, exists)

			count, err = db.NewSelect().TableExpr("payment_initiation_related_payments").Count(ctx)
			require.NoError(t, err)
			require.Equal(t, 1, count)

			exists, err = db.NewSelect().TableExpr("payment_initiation_related_payments").
				Where("payment_initiation_id = ?", "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0").
				Where("payment_id = ?", "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0IiwiVHlwZSI6IlBBWS1JTiJ9").
				Exists(ctx)
			require.NoError(t, err)
			require.True(t, exists)

			count, err = db.NewSelect().TableExpr("payment_initiation_adjustments").Count(ctx)
			require.NoError(t, err)
			// We should have only one, since the second one should have been skipped because of
			// a status that disappeared in v3
			require.Equal(t, 1, count)

			tID := models.MustPaymentInitiationIDFromString("eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0")
			createdAt, err := time.Parse(time.RFC3339Nano, "2025-01-06T13:30:02.854346Z")
			require.NoError(t, err)

			exists, err = db.NewSelect().TableExpr("payment_initiation_adjustments").
				Where("id = ?", models.PaymentInitiationAdjustmentID{
					PaymentInitiationID: *tID,
					CreatedAt:           createdAt.UTC(),
					Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION,
				}).
				Exists(ctx)
			require.NoError(t, err)
			require.True(t, exists)
		},
	}
}

func testPoolsMigrations() testmigrations.Hook {
	return testmigrations.Hook{
		After: func(ctx context.Context, t *testing.T, db bun.IDB) {
			count, err := db.NewSelect().TableExpr("pools").Count(ctx)
			require.NoError(t, err)
			require.Equal(t, 1, count)

			exists, err := db.NewSelect().TableExpr("pools").
				Where("id = ?", "83064af3-bb81-4514-a6d4-afba340825ce").
				Exists(ctx)
			require.NoError(t, err)
			require.True(t, exists)

			count, err = db.NewSelect().TableExpr("pool_accounts").Count(ctx)
			require.NoError(t, err)
			require.Equal(t, 1, count)

			exists, err = db.NewSelect().TableExpr("pool_accounts").
				Where("pool_id = ?", "83064af3-bb81-4514-a6d4-afba340825ce").
				Where("account_id = ?", "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0").
				Exists(ctx)
			require.NoError(t, err)
			require.True(t, exists)
		},
	}
}

func testPaymentReversalsMigrations() testmigrations.Hook {
	return testmigrations.Hook{
		After: func(ctx context.Context, t *testing.T, db bun.IDB) {
			count, err := db.NewSelect().TableExpr("payment_initiation_reversals").Count(ctx)
			require.NoError(t, err)
			require.Equal(t, 1, count)

			exists, err := db.NewSelect().TableExpr("payment_initiation_reversals").
				Where("id = ?", "eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0X3JldmVyc2FsIn0").
				Exists(ctx)
			require.NoError(t, err)
			require.True(t, exists)

			count, err = db.NewSelect().TableExpr("payment_initiation_reversal_adjustments").Count(ctx)
			require.NoError(t, err)
			require.Equal(t, 2, count)

			reversalID := models.MustPaymentInitiationReversalIDFromString("eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0X3JldmVyc2FsIn0")
			createdAt, err := time.Parse(time.RFC3339Nano, "2025-01-06T14:40:02.854346Z")
			require.NoError(t, err)

			exists, err = db.NewSelect().TableExpr("payment_initiation_reversal_adjustments").
				Where("id = ?", models.PaymentInitiationReversalAdjustmentID{
					PaymentInitiationReversalID: *reversalID,
					CreatedAt:                   createdAt.UTC(),
					Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSING,
				}).
				Exists(ctx)
			require.NoError(t, err)
			require.True(t, exists)

			exists, err = db.NewSelect().TableExpr("payment_initiation_reversal_adjustments").
				Where("id = ?", models.PaymentInitiationReversalAdjustmentID{
					PaymentInitiationReversalID: *reversalID,
					CreatedAt:                   createdAt.UTC(),
					Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
				}).
				Exists(ctx)
			require.NoError(t, err)
			require.True(t, exists)
		},
	}
}

func testReferenceConnectorMigrations() testmigrations.Hook {
	return testmigrations.Hook{
		After: func(ctx context.Context, t *testing.T, db bun.IDB) {
			for _, connectorID := range []string{
				"eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9",
				"eyJQcm92aWRlciI6ImFkeWVuIiwiUmVmZXJlbmNlIjoiNGEwNzUyYWUtYWIxYS00NWI4LTgyNzItMjI2ODA3OTE2NTQ0In0",
				"eyJQcm92aWRlciI6ImF0bGFyIiwiUmVmZXJlbmNlIjoiN2JkZTk4NGUtYzY1OC00MzNiLWE1OGEtZTUxMGMwMTYwMDYwIn0",
				"eyJQcm92aWRlciI6ImJhbmtpbmdjaXJjbGUiLCJSZWZlcmVuY2UiOiIwODQ4OWFlNC0zOGUxLTQwMTEtYjViMS1mZjkxMTliYWEzNDkifQ",
				"eyJQcm92aWRlciI6ImN1cnJlbmN5Y2xvdWQiLCJSZWZlcmVuY2UiOiJlNmI4OGFlZS05OTI0LTQ4ZmYtYTZkMS1mYmIwZjJjMjRkYWYifQ",
				"eyJQcm92aWRlciI6ImdlbmVyaWMiLCJSZWZlcmVuY2UiOiIwYmE0MDNiYi0zYzlmLTQ2OTUtYmQxZC0yYmQ5ZDdiMjgwOTQifQ",
				"eyJQcm92aWRlciI6Im1hbmdvcGF5IiwiUmVmZXJlbmNlIjoiZTQ0MGIyMzgtM2RkNi00YzhlLTk5MDktZTJjOTgzODA2MTgyIn0",
				"eyJQcm92aWRlciI6Im1vZHVsciIsIlJlZmVyZW5jZSI6IjYzZTZlNDIyLWQ5MWMtNDQ3YS1hODU0LTE5ODJkYTU1YzljYyJ9",
				"eyJQcm92aWRlciI6InN0cmlwZSIsIlJlZmVyZW5jZSI6ImIwYzZjNTdhLTM3MDYtNDRmMi1iMDdmLTE3YjNiYTdhZDhkYyJ9",
				"eyJQcm92aWRlciI6Indpc2UiLCJSZWZlcmVuY2UiOiI4OWJlZDg1MS1kMjIyLTQ2NzItYjEwYy00ZDczZWE2ZGY0NGEifQ",
			} {
				id := models.MustConnectorIDFromString(connectorID)

				exists, err := db.NewSelect().TableExpr("connectors").
					Where("id = ?", connectorID).
					Where("reference = ?", id.Reference).
					Exists(ctx)
				require.NoError(t, err)
				require.True(t, exists)
			}
		},
	}
}
