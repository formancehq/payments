package storage

import (
	"context"
	"testing"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/query"
	"github.com/formancehq/go-libs/v2/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	poolID1 = uuid.New()
	poolID2 = uuid.New()
	poolID3 = uuid.New()
)

func defaultPools() []models.Pool {
	defaultAccounts := defaultAccounts()
	return []models.Pool{
		{
			ID:           poolID1,
			Name:         "test1",
			CreatedAt:    now.Add(-60 * time.Minute).UTC().Time,
			PoolAccounts: []models.AccountID{defaultAccounts[0].ID, defaultAccounts[1].ID},
		},
		{
			ID:           poolID2,
			Name:         "test2",
			CreatedAt:    now.Add(-30 * time.Minute).UTC().Time,
			PoolAccounts: []models.AccountID{defaultAccounts[2].ID},
		},
		{
			ID:           poolID3,
			Name:         "test3",
			CreatedAt:    now.Add(-55 * time.Minute).UTC().Time,
			PoolAccounts: []models.AccountID{defaultAccounts[2].ID},
		},
	}
}

func upsertPool(t *testing.T, ctx context.Context, storage Storage, pool models.Pool) {
	require.NoError(t, storage.PoolsUpsert(ctx, pool))
}

func TestPoolsUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPool(t, ctx, store, defaultPools()[0])
	upsertPool(t, ctx, store, defaultPools()[1])

	t.Run("upsert with same name", func(t *testing.T) {
		poolID3 := uuid.New()
		p := models.Pool{
			ID:           poolID3,
			Name:         "test1",
			CreatedAt:    now.Add(-30 * time.Minute).UTC().Time,
			PoolAccounts: []models.AccountID{defaultAccounts()[2].ID},
		}

		err := store.PoolsUpsert(ctx, p)
		require.Error(t, err)
	})

	t.Run("upsert with same id", func(t *testing.T) {
		upsertPool(t, ctx, store, defaultPools()[1])

		actual, err := store.PoolsGet(ctx, defaultPools()[1].ID)
		require.NoError(t, err)
		require.Equal(t, defaultPools()[1], *actual)
	})

	t.Run("upsert with same id but more related accounts", func(t *testing.T) {
		p := defaultPools()[0]
		p.PoolAccounts = append(p.PoolAccounts, defaultAccounts()[2].ID)

		upsertPool(t, ctx, store, p)

		actual, err := store.PoolsGet(ctx, defaultPools()[0].ID)
		require.NoError(t, err)
		require.Equal(t, p, *actual)
	})

	t.Run("upsert but account does not exist", func(t *testing.T) {
		p := defaultPools()[0]
		p.PoolAccounts = append(p.PoolAccounts, models.AccountID{
			Reference:   "unknown",
			ConnectorID: defaultConnector.ID,
		})

		err := store.PoolsUpsert(ctx, p)
		require.Error(t, err)
	})
}

func TestPoolsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPool(t, ctx, store, defaultPools()[0])
	upsertPool(t, ctx, store, defaultPools()[1])
	upsertPool(t, ctx, store, defaultPools()[2])

	t.Run("get existing pool", func(t *testing.T) {
		for _, p := range defaultPools() {
			actual, err := store.PoolsGet(ctx, p.ID)
			require.NoError(t, err)
			require.Equal(t, p, *actual)
		}
	})

	t.Run("get non-existing pool", func(t *testing.T) {
		p, err := store.PoolsGet(ctx, uuid.New())
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, p)
	})
}

func TestPoolsDelete(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPool(t, ctx, store, defaultPools()[0])
	upsertPool(t, ctx, store, defaultPools()[1])
	upsertPool(t, ctx, store, defaultPools()[2])

	t.Run("delete unknown pool", func(t *testing.T) {
		deleted, err := store.PoolsDelete(ctx, uuid.New())
		require.NoError(t, err)
		require.False(t, deleted)
		for _, p := range defaultPools() {
			actual, err := store.PoolsGet(ctx, p.ID)
			require.NoError(t, err)
			require.Equal(t, p, *actual)
		}
	})

	t.Run("delete existing pool", func(t *testing.T) {
		deleted, err := store.PoolsDelete(ctx, defaultPools()[0].ID)
		require.NoError(t, err)
		require.True(t, deleted)

		_, err = store.PoolsGet(ctx, defaultPools()[0].ID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)

		actual, err := store.PoolsGet(ctx, defaultPools()[1].ID)
		require.NoError(t, err)
		require.Equal(t, defaultPools()[1], *actual)
	})
}

func TestPoolsAddAccount(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPool(t, ctx, store, defaultPools()[0])
	upsertPool(t, ctx, store, defaultPools()[1])

	t.Run("add account to unknown pool", func(t *testing.T) {
		err := store.PoolsAddAccount(ctx, uuid.New(), defaultAccounts()[0].ID)
		require.Error(t, err)
	})

	t.Run("add account to pool", func(t *testing.T) {
		require.NoError(t, store.PoolsAddAccount(ctx, defaultPools()[0].ID, defaultAccounts()[2].ID))

		p := defaultPools()[0]
		p.PoolAccounts = append(p.PoolAccounts, defaultAccounts()[2].ID)

		actual, err := store.PoolsGet(ctx, defaultPools()[0].ID)
		require.NoError(t, err)
		require.Equal(t, p, *actual)
	})

	t.Run("add account to pool but account does not exist", func(t *testing.T) {
		err := store.PoolsAddAccount(ctx, defaultPools()[0].ID, models.AccountID{
			Reference:   "unknown",
			ConnectorID: defaultConnector.ID,
		})
		require.Error(t, err)
	})
}

func TestPoolsRemoveAccount(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPool(t, ctx, store, defaultPools()[0])
	upsertPool(t, ctx, store, defaultPools()[1])

	t.Run("remove unknown account from pool", func(t *testing.T) {
		require.NoError(t, store.PoolsRemoveAccount(ctx, defaultPools()[0].ID, models.AccountID{
			Reference:   "unknown",
			ConnectorID: defaultConnector.ID,
		}))
	})

	t.Run("remove account from unknown pool", func(t *testing.T) {
		require.NoError(t, store.PoolsRemoveAccount(ctx, uuid.New(), defaultAccounts()[0].ID))
	})

	t.Run("remove account from pool", func(t *testing.T) {
		require.NoError(t, store.PoolsRemoveAccount(ctx, defaultPools()[0].ID, defaultAccounts()[1].ID))

		p := defaultPools()[0]
		p.PoolAccounts = p.PoolAccounts[:1]

		actual, err := store.PoolsGet(ctx, defaultPools()[0].ID)
		require.NoError(t, err)
		require.Equal(t, p, *actual)
	})
}

func TestPoolsRemoveAccountFromConnectorID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPool(t, ctx, store, defaultPools()[0])

	t.Run("remove accounts from unknown connector", func(t *testing.T) {
		require.NoError(t, store.PoolsRemoveAccountsFromConnectorID(ctx, models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}))

		actual, err := store.PoolsGet(ctx, defaultPools()[0].ID)
		require.NoError(t, err)
		require.Equal(t, defaultPools()[0], *actual)
	})

	t.Run("remove accounts from connector", func(t *testing.T) {
		require.NoError(t, store.PoolsRemoveAccountsFromConnectorID(ctx, defaultConnector.ID))

		actual, err := store.PoolsGet(ctx, defaultPools()[0].ID)
		require.NoError(t, err)
		require.Len(t, actual.PoolAccounts, 0)
	})
}

func TestPoolsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPool(t, ctx, store, defaultPools()[0])
	upsertPool(t, ctx, store, defaultPools()[1])
	upsertPool(t, ctx, store, defaultPools()[2])

	t.Run("wrong query builder operator when listing by name", func(t *testing.T) {
		q := NewListPoolsQuery(
			bunpaginate.NewPaginatedQueryOptions(PoolQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("name", "test1")),
		)

		cursor, err := store.PoolsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
		assert.True(t, errors.Is(err, ErrValidation))
		assert.Regexp(t, "name", err.Error())
	})

	t.Run("list pools by name", func(t *testing.T) {
		q := NewListPoolsQuery(
			bunpaginate.NewPaginatedQueryOptions(PoolQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("name", "test1")),
		)

		cursor, err := store.PoolsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		require.Equal(t, []models.Pool{defaultPools()[0]}, cursor.Data)
	})

	t.Run("list pools by unknown name", func(t *testing.T) {
		q := NewListPoolsQuery(
			bunpaginate.NewPaginatedQueryOptions(PoolQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("name", "unknown")),
		)

		cursor, err := store.PoolsList(ctx, q)
		require.NoError(t, err)
		require.Empty(t, cursor.Data)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
	})

	t.Run("list pools by id", func(t *testing.T) {
		q := NewListPoolsQuery(
			bunpaginate.NewPaginatedQueryOptions(PoolQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", poolID1.String())),
		)

		cursor, err := store.PoolsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		require.Equal(t, []models.Pool{defaultPools()[0]}, cursor.Data)
	})

	t.Run("list pools by unknown id", func(t *testing.T) {
		q := NewListPoolsQuery(
			bunpaginate.NewPaginatedQueryOptions(PoolQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", uuid.New().String())),
		)

		cursor, err := store.PoolsList(ctx, q)
		require.NoError(t, err)
		require.Empty(t, cursor.Data)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
	})

	t.Run("list pools by account id 1", func(t *testing.T) {
		q := NewListPoolsQuery(
			bunpaginate.NewPaginatedQueryOptions(PoolQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("account_id", defaultAccounts()[0].ID.String())),
		)

		cursor, err := store.PoolsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		require.Equal(t, []models.Pool{defaultPools()[0]}, cursor.Data)
	})

	t.Run("list pools by account id 2", func(t *testing.T) {
		q := NewListPoolsQuery(
			bunpaginate.NewPaginatedQueryOptions(PoolQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("account_id", defaultAccounts()[2].ID.String())),
		)

		cursor, err := store.PoolsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		require.Equal(t, []models.Pool{defaultPools()[1], defaultPools()[2]}, cursor.Data)
	})

	t.Run("list pools by unknown account id", func(t *testing.T) {
		q := NewListPoolsQuery(
			bunpaginate.NewPaginatedQueryOptions(PoolQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("account_id", uuid.New().String())),
		)

		cursor, err := store.PoolsList(ctx, q)
		require.NoError(t, err)
		require.Empty(t, cursor.Data)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
	})

	t.Run("unknown query builder key when listing", func(t *testing.T) {
		q := NewListPoolsQuery(
			bunpaginate.NewPaginatedQueryOptions(PoolQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("unknown", "test1")),
		)

		cursor, err := store.PoolsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list pools test cursor", func(t *testing.T) {
		q := NewListPoolsQuery(
			bunpaginate.NewPaginatedQueryOptions(PoolQuery{}).
				WithPageSize(1),
		)
		defaultPools := defaultPools()

		cursor, err := store.PoolsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		require.Equal(t, []models.Pool{defaultPools[1]}, cursor.Data)

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PoolsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		require.Equal(t, []models.Pool{defaultPools[2]}, cursor.Data)

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PoolsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		require.Equal(t, []models.Pool{defaultPools[0]}, cursor.Data)

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PoolsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		require.Equal(t, []models.Pool{defaultPools[2]}, cursor.Data)

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PoolsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		require.Equal(t, []models.Pool{defaultPools[1]}, cursor.Data)
	})
}
