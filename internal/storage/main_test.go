package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"os"
	"testing"

	"github.com/formancehq/go-libs/v3/bun/bunconnect"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/testing/docker"
	"github.com/formancehq/go-libs/v3/testing/platform/pgtesting"
	"github.com/formancehq/go-libs/v3/testing/utils"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

var (
	srv   *pgtesting.PostgresServer
	bunDB *bun.DB
)

func TestMain(m *testing.M) {
	utils.WithTestMain(func(t *utils.TestingTForMain) int {
		srv = pgtesting.CreatePostgresServer(t, docker.NewPool(t, logging.Testing()))

		db, err := sql.Open("pgx", srv.GetDSN())
		if err != nil {
			logging.Error(err)
			os.Exit(1)
		}

		bunDB = bun.NewDB(db, pgdialect.New())

		//return m.Run()
		code := m.Run()

		// Ensure the global bunDB is closed at the end of the test suite
		_ = bunDB.Close()
		return code
	})
}

func newStore(t *testing.T) Storage {
	t.Helper()
	ctx := logging.TestingContext()

	pgServer := srv.NewDatabase(t)

	db, err := bunconnect.OpenSQLDB(ctx, pgServer.ConnectionOptions())
	require.NoError(t, err)

	key := make([]byte, 64)
	_, err = rand.Read(key)
	require.NoError(t, err)

	err = Migrate(context.Background(), logging.Testing(), db, "test")
	require.NoError(t, err)

	//return newStorage(logging.Testing(), db, string(key))

	st := newStorage(logging.Testing(), db, string(key))

	// Ensure the store (and its DB connections) are closed before the per-test database is dropped.
	t.Cleanup(func() {
		_ = st.Close()
	})

	return st
}

func cleanupOutboxHelper(ctx context.Context, store Storage) func() {
	return func() {
		pendingEvents, _ := store.OutboxEventsPollPending(ctx, 1000)
		var eventIDs []models.EventID
		var eventsSent []models.EventSent
		for _, event := range pendingEvents {
			eventIDs = append(eventIDs, event.ID)
			eventsSent = append(eventsSent, models.EventSent{})
		}
		if len(eventIDs) > 0 {
			_ = store.OutboxEventsDeleteAndRecordSent(ctx, eventIDs, eventsSent)
		}
	}
}
