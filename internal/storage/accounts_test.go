package storage

import (
	"context"
	"testing"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/go-libs/v2/query"
	"github.com/formancehq/go-libs/v2/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func defaultAccounts() []models.Account {
	return []models.Account{
		{
			ID: models.AccountID{
				Reference:   "test1",
				ConnectorID: defaultConnector.ID,
			},
			ConnectorID:  defaultConnector.ID,
			Reference:    "test1",
			CreatedAt:    now.Add(-60 * time.Minute).UTC().Time,
			Type:         models.ACCOUNT_TYPE_INTERNAL,
			Name:         pointer.For("test1"),
			DefaultAsset: pointer.For("USD/2"),
			Metadata: map[string]string{
				"foo": "bar",
			},
			Raw: []byte(`{}`),
		},
		{
			ID: models.AccountID{
				Reference:   "test2",
				ConnectorID: defaultConnector.ID,
			},
			ConnectorID: defaultConnector.ID,
			Reference:   "test2",
			CreatedAt:   now.Add(-30 * time.Minute).UTC().Time,
			Type:        models.ACCOUNT_TYPE_INTERNAL,
			Metadata: map[string]string{
				"foo2": "bar2",
			},
			Raw: []byte(`{}`),
		},
		{
			ID: models.AccountID{
				Reference:   "test3",
				ConnectorID: defaultConnector.ID,
			},
			ConnectorID: defaultConnector.ID,
			Reference:   "test3",
			CreatedAt:   now.Add(-45 * time.Minute).UTC().Time,
			Type:        models.ACCOUNT_TYPE_EXTERNAL,
			Name:        pointer.For("test3"),
			Metadata: map[string]string{
				"foo3": "bar3",
			},
			Raw: []byte(`{}`),
		},
	}
}

func defaultAccounts2() []models.Account {
	return []models.Account{
		{
			ID: models.AccountID{
				Reference:   "test1",
				ConnectorID: defaultConnector2.ID,
			},
			ConnectorID:  defaultConnector2.ID,
			Reference:    "test1",
			CreatedAt:    now.Add(-55 * time.Minute).UTC().Time,
			Type:         models.ACCOUNT_TYPE_INTERNAL,
			Name:         pointer.For("test1"),
			DefaultAsset: pointer.For("USD/2"),
			Metadata: map[string]string{
				"foo5": "bar5",
			},
			Raw: []byte(`{}`),
		},
	}
}

func defaultAccounts3() []models.Account {
	createdAt := time.Now().UTC().Truncate(time.Minute).Add(-2 * time.Minute).UTC()
	return []models.Account{
		{
			ID: models.AccountID{
				Reference:   "sort-test",
				ConnectorID: defaultConnector2.ID,
			},
			ConnectorID:  defaultConnector2.ID,
			Reference:    "sort-test",
			CreatedAt:    createdAt,
			Type:         models.ACCOUNT_TYPE_INTERNAL,
			Name:         pointer.For("sort-test"),
			DefaultAsset: pointer.For("EUR/2"),
			Metadata: map[string]string{
				"unrelated": "keyval",
			},
			Raw: []byte(`{}`),
		},
		{
			ID: models.AccountID{
				Reference:   "sort-test2",
				ConnectorID: defaultConnector2.ID,
			},
			ConnectorID:  defaultConnector2.ID,
			Reference:    "sort-test2",
			CreatedAt:    createdAt,
			Type:         models.ACCOUNT_TYPE_INTERNAL,
			Name:         pointer.For("sort-test2"),
			DefaultAsset: pointer.For("EUR/2"),
			Metadata: map[string]string{
				"metadata": "keyval",
			},
			Raw: []byte(`{}`),
		},
	}
}

func upsertAccounts(t *testing.T, ctx context.Context, storage Storage, accounts []models.Account) {
	require.NoError(t, storage.AccountsUpsert(ctx, accounts))
}

func TestAccountsUpsert(t *testing.T) {
	t.Parallel()

	now := time.Now()
	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())

	t.Run("upsert empty list", func(t *testing.T) {
		require.NoError(t, store.AccountsUpsert(ctx, []models.Account{}))
	})

	t.Run("same id insert", func(t *testing.T) {
		id := models.AccountID{
			Reference:   "test1",
			ConnectorID: defaultConnector.ID,
		}

		// Same account I but different fields
		acc := models.Account{
			ID:           id,
			ConnectorID:  defaultConnector.ID,
			Reference:    "test1",
			CreatedAt:    now.Add(-12 * time.Minute).UTC().Time,
			Type:         models.ACCOUNT_TYPE_EXTERNAL,
			Name:         pointer.For("changed"),
			DefaultAsset: pointer.For("EUR"),
			Metadata: map[string]string{
				"foo4": "bar4",
			},
			Raw: []byte(`{}`),
		}

		require.NoError(t, store.AccountsUpsert(ctx, []models.Account{acc}))

		// Check that account was not updated
		account, err := store.AccountsGet(ctx, id)
		require.NoError(t, err)

		// Accounts should not have changed
		require.Equal(t, defaultAccounts()[0], *account)
	})

	t.Run("unknown connector id", func(t *testing.T) {
		unknownConnectorID := models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}

		acc := models.Account{
			ID: models.AccountID{
				Reference:   "test_unknown",
				ConnectorID: unknownConnectorID,
			},
			ConnectorID:  unknownConnectorID,
			Reference:    "test_unknown",
			CreatedAt:    now.Add(-12 * time.Minute).UTC().Time,
			Type:         models.ACCOUNT_TYPE_EXTERNAL,
			Name:         pointer.For("changed"),
			DefaultAsset: pointer.For("EUR"),
			Metadata: map[string]string{
				"foo4": "bar4",
			},
			Raw: []byte(`{}`),
		}

		require.Error(t, store.AccountsUpsert(ctx, []models.Account{acc}))
	})
}

func TestAccountsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())

	t.Run("get account", func(t *testing.T) {
		for _, acc := range defaultAccounts() {
			account, err := store.AccountsGet(ctx, acc.ID)
			require.NoError(t, err)
			require.Equal(t, acc, *account)
		}
	})

	t.Run("get unknown account", func(t *testing.T) {
		acc := models.AccountID{
			Reference:   "unknown",
			ConnectorID: defaultConnector.ID,
		}

		account, err := store.AccountsGet(ctx, acc)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, account)
	})
}

func TestAccountsDelete(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertAccounts(t, ctx, store, defaultAccounts2())

	t.Run("delete account from unknown connector", func(t *testing.T) {
		unknownConnectorID := models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}

		require.NoError(t, store.AccountsDeleteFromConnectorID(ctx, unknownConnectorID))

		for _, acc := range defaultAccounts() {
			account, err := store.AccountsGet(ctx, acc.ID)
			require.NoError(t, err)
			require.Equal(t, acc, *account)
		}

		for _, acc := range defaultAccounts2() {
			account, err := store.AccountsGet(ctx, acc.ID)
			require.NoError(t, err)
			require.Equal(t, acc, *account)
		}
	})

	t.Run("delete account from default connector", func(t *testing.T) {
		require.NoError(t, store.AccountsDeleteFromConnectorID(ctx, defaultConnector.ID))

		for _, acc := range defaultAccounts() {
			account, err := store.AccountsGet(ctx, acc.ID)
			require.Error(t, err)
			require.Nil(t, account)
			require.ErrorIs(t, err, ErrNotFound)
		}

		for _, acc := range defaultAccounts2() {
			account, err := store.AccountsGet(ctx, acc.ID)
			require.NoError(t, err)
			require.Equal(t, acc, *account)
		}
	})

}

func TestAccountsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertAccounts(t, ctx, store, defaultAccounts2())
	upsertAccounts(t, ctx, store, defaultAccounts3())

	t.Run("list accounts by reference", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("reference", "test1")),
		)

		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		require.Equal(t, defaultAccounts2()[0], cursor.Data[0])
		require.Equal(t, defaultAccounts()[0], cursor.Data[1])
	})

	t.Run("list accounts by reference 2", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("reference", "test2")),
		)
		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Equal(t, defaultAccounts()[1], cursor.Data[0])
	})

	t.Run("list accounts by unknown reference", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("reference", "unknown")),
		)

		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list accounts by connector id", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connector_id", defaultConnector.ID)),
		)
		accounts := defaultAccounts()
		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 3)
		require.False(t, cursor.HasMore)
		require.Equal(t, accounts[1], cursor.Data[0])
		require.Equal(t, accounts[2], cursor.Data[1])
		require.Equal(t, accounts[0], cursor.Data[2])
	})

	t.Run("list accounts by connector id 2", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connector_id", defaultConnector2.ID)),
		)
		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 3)
		require.False(t, cursor.HasMore)
		require.Equal(t, defaultAccounts3()[1], cursor.Data[0])
		require.Equal(t, defaultAccounts3()[0], cursor.Data[1])
		require.Equal(t, defaultAccounts2()[0], cursor.Data[2])
	})

	t.Run("list accounts by unknown connector id", func(t *testing.T) {
		unknownConnectorID := models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}

		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connector_id", unknownConnectorID)),
		)
		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list accounts by type", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("type", models.ACCOUNT_TYPE_INTERNAL)),
		)
		accounts := defaultAccounts()
		accounts2 := defaultAccounts2()
		accounts3 := defaultAccounts3()

		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 5)
		require.False(t, cursor.HasMore)
		require.Equal(t, accounts3[1], cursor.Data[0])
		require.Equal(t, accounts3[0], cursor.Data[1])
		require.Equal(t, accounts[1], cursor.Data[2])
		require.Equal(t, accounts2[0], cursor.Data[3])
		require.Equal(t, accounts[0], cursor.Data[4])
	})

	t.Run("list accounts by unknown type", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("type", "unknown")),
		)

		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list accounts by default asset", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("default_asset", "USD/2")),
		)
		accounts := defaultAccounts()
		accounts2 := defaultAccounts2()

		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		require.Equal(t, accounts2[0], cursor.Data[0])
		require.Equal(t, accounts[0], cursor.Data[1])
	})

	t.Run("list accounts by unknown default asset", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("default_asset", "unknown")),
		)

		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list accounts by name", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("name", "test1")),
		)

		accounts := defaultAccounts()
		accounts2 := defaultAccounts2()
		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		require.Equal(t, accounts2[0], cursor.Data[0])
		require.Equal(t, accounts[0], cursor.Data[1])
	})

	t.Run("list accounts by name 2", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("name", "test3")),
		)

		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Equal(t, defaultAccounts()[2], cursor.Data[0])
	})

	t.Run("list accounts by unknown name", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("name", "unknown")),
		)

		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list accounts by metadata", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[foo]", "bar")),
		)

		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Equal(t, defaultAccounts()[0], cursor.Data[0])
	})

	t.Run("list accounts by unknown metadata", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[foo]", "unknown")),
		)

		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("wrong query builder operator with metadata", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("metadata[foo]", "unknown")),
		)

		cursor, err := store.AccountsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("query builder unknown key", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("unknown", "unknown")),
		)

		cursor, err := store.AccountsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list accounts test cursor", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(1),
		)
		accounts := defaultAccounts()
		accounts2 := defaultAccounts2()
		accounts3 := defaultAccounts3()

		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Next)
		require.Empty(t, cursor.Previous)
		require.Equal(t, accounts3[1], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Next)
		require.NotEmpty(t, cursor.Previous)
		require.Equal(t, accounts3[0], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Next)
		require.NotEmpty(t, cursor.Previous)
		require.Equal(t, accounts[1], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Next)
		require.NotEmpty(t, cursor.Previous)
		require.Equal(t, accounts[2], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Next)
		require.NotEmpty(t, cursor.Previous)
		require.Equal(t, accounts2[0], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Next)
		require.NotEmpty(t, cursor.Previous)
		require.Equal(t, accounts[0], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Next)
		require.NotEmpty(t, cursor.Previous)
		require.Equal(t, accounts2[0], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Next)
		require.NotEmpty(t, cursor.Previous)
		require.Equal(t, accounts[2], cursor.Data[0])
	})
}
