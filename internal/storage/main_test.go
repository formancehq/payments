package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"os"
	"testing"

	"github.com/formancehq/go-libs/v2/bun/bunconnect"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/testing/docker"
	"github.com/formancehq/go-libs/v2/testing/platform/pgtesting"
	"github.com/formancehq/go-libs/v2/testing/utils"
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

		return m.Run()
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

	err = Migrate(context.Background(), db)
	require.NoError(t, err)

	return newStorage(logging.Testing(), db, string(key))
}
