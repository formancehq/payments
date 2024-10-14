package storage

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/formancehq/go-libs/testing/docker"
	"github.com/formancehq/go-libs/testing/utils"

	"github.com/formancehq/go-libs/logging"
	"github.com/formancehq/go-libs/testing/platform/pgtesting"
	migrationstorage "github.com/formancehq/payments/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

var (
	srv *pgtesting.PostgresServer
)

func TestMain(m *testing.M) {
	utils.WithTestMain(func(t *utils.TestingTForMain) int {
		srv = pgtesting.CreatePostgresServer(t, docker.NewPool(t, logging.Testing()))

		return m.Run()
	})
}

func newStore(t *testing.T) *Storage {
	t.Helper()

	pgServer := srv.NewDatabase(t)

	config, err := pgx.ParseConfig(pgServer.ConnString())
	require.NoError(t, err)

	key := make([]byte, 64)
	_, err = rand.Read(key)
	require.NoError(t, err)

	db := bun.NewDB(stdlib.OpenDB(*config), pgdialect.New())
	t.Cleanup(func() {
		_ = db.Close()
	})

	err = migrationstorage.Migrate(context.Background(), db)
	require.NoError(t, err)

	store := NewStorage(
		db,
		string(key),
	)

	return store
}