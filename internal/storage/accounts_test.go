package storage

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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
			Connector:    defaultConnector.Base(),
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
			Connector:   defaultConnector.Base(),
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
			Connector:   defaultConnector.Base(),
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
			Connector:    defaultConnector2.Base(),
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
			Connector:    defaultConnector2.Base(),
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
			Connector:    defaultConnector2.Base(),
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
	defer store.Close()

	cleanupOutbox := func() {
		// Get all pending events for the default connector and delete them
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 1000)
		require.NoError(t, err)
		for _, event := range pendingEvents {
			if event.ConnectorID != nil && *event.ConnectorID == defaultConnector.ID {
				// Create a dummy EventSent for deletion
				eventSent := models.EventSent{
					ID: models.EventID{
						EventIdempotencyKey: "cleanup",
						ConnectorID:         event.ConnectorID,
					},
					ConnectorID: event.ConnectorID,
					SentAt:      now.UTC().Time,
				}
				// Delete using the proper method
				err = store.OutboxEventsDeleteAndRecordSent(ctx, event.ID, eventSent)
				require.NoError(t, err)
			}
		}
	}

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	cleanupOutbox() // Remove the outbox events created by updserting accounts

	t.Run("upsert empty list", func(t *testing.T) {
		require.NoError(t, store.AccountsUpsert(ctx, []models.Account{}))
	})

	t.Run("same id insert", func(t *testing.T) {
		defer cleanupOutbox() // Also clean up after test

		id := models.AccountID{
			Reference:   "test1",
			ConnectorID: defaultConnector.ID,
		}

		// Same account I but different fields
		acc := models.Account{
			ID:           id,
			ConnectorID:  defaultConnector.ID,
			Connector:    defaultConnector.Base(),
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

		// Verify outbox events were not created (account already existed)
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 10)
		require.NoError(t, err)
		// Should have no new events since the account already existed
		assert.Len(t, pendingEvents, 0, "No outbox events should be created for existing accounts")
	})

	t.Run("unknown connector id", func(t *testing.T) {
		defer cleanupOutbox()

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

		// Verify outbox events were not created
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 10)
		require.NoError(t, err)
		require.Len(t, pendingEvents, 0)
	})

	t.Run("outbox events created for new accounts", func(t *testing.T) {
		defer cleanupOutbox()

		// Create new accounts
		newAccounts := []models.Account{
			{
				ID: models.AccountID{
					Reference:   "outbox-test-1",
					ConnectorID: defaultConnector.ID,
				},
				ConnectorID:  defaultConnector.ID,
				Connector:    defaultConnector.Base(),
				Reference:    "outbox-test-1",
				CreatedAt:    time.Now().UTC().Time,
				Type:         models.ACCOUNT_TYPE_INTERNAL,
				Name:         pointer.For("Outbox Test 1"),
				DefaultAsset: pointer.For("USD/2"),
				Metadata: map[string]string{
					"test": "outbox",
				},
				Raw: []byte(`{}`),
			},
			{
				ID: models.AccountID{
					Reference:   "outbox-test-2",
					ConnectorID: defaultConnector.ID,
				},
				ConnectorID:  defaultConnector.ID,
				Connector:    defaultConnector.Base(),
				Reference:    "outbox-test-2",
				CreatedAt:    time.Now().UTC().Time,
				Type:         models.ACCOUNT_TYPE_EXTERNAL,
				Name:         pointer.For("Outbox Test 2"),
				DefaultAsset: pointer.For("EUR/2"),
				Metadata: map[string]string{
					"test": "outbox",
				},
				Raw: []byte(`{}`),
			},
		}

		// Create a set of expected idempotency keys
		expectedKeys := make(map[string]bool)
		for _, account := range newAccounts {
			expectedKeys[account.IdempotencyKey()] = true
		}

		// Insert accounts
		require.NoError(t, store.AccountsUpsert(ctx, newAccounts))

		// Verify outbox events were created
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		// Filter events to only those we just created
		ourEvents := make([]models.OutboxEvent, 0)
		for _, event := range pendingEvents {
			if event.EventType == "account.saved" && expectedKeys[event.IdempotencyKey] {
				ourEvents = append(ourEvents, event)
			}
		}
		require.Len(t, ourEvents, 2, "expected 2 outbox events for 2 new accounts")

		// Create a map of expected accounts by idempotency key for easier lookup
		expectedAccountsByKey := make(map[string]models.Account)
		for _, account := range newAccounts {
			expectedAccountsByKey[account.IdempotencyKey()] = account
		}

		// Check event details
		for _, event := range ourEvents {
			assert.Equal(t, "account.saved", event.EventType)
			assert.Equal(t, models.OUTBOX_STATUS_PENDING, event.Status)
			assert.Equal(t, defaultConnector.ID, *event.ConnectorID)
			assert.Equal(t, 0, event.RetryCount)
			assert.Nil(t, event.Error)
			assert.NotEqual(t, uuid.Nil, event.ID)
			assert.NotEmpty(t, event.IdempotencyKey)

			// Find the matching account by idempotency key
			expectedAccount, found := expectedAccountsByKey[event.IdempotencyKey]
			require.True(t, found, "event idempotency key should match one of the accounts")

			// Verify payload contains account data
			var payload map[string]interface{}
			err := json.Unmarshal(event.Payload, &payload)
			require.NoError(t, err)
			assert.Equal(t, expectedAccount.ID.String(), payload["id"])
			assert.Contains(t, payload, "name")
			assert.Contains(t, payload, "type")
			assert.Equal(t, expectedAccount.ConnectorID.String(), payload["connectorID"])

			// Verify EntityID matches account ID
			assert.Equal(t, expectedAccount.ID.String(), event.EntityID)

			// Verify idempotency key matches account
			assert.Equal(t, expectedAccount.IdempotencyKey(), event.IdempotencyKey)
		}
	})

	t.Run("no outbox events for existing accounts", func(t *testing.T) {
		defer cleanupOutbox()

		// Try to insert existing accounts (should not create outbox events due to ON CONFLICT DO NOTHING)
		require.NoError(t, store.AccountsUpsert(ctx, defaultAccounts()))

		// Verify no outbox events were created
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 10)
		require.NoError(t, err)
		assert.Len(t, pendingEvents, 0)
	})

	t.Run("rollback on foreign key violation", func(t *testing.T) {
		defer cleanupOutbox()

		upsertConnector(t, ctx, store, defaultConnector)

		// Count existing accounts
		accountsBefore, err := store.AccountsList(ctx, NewListAccountsQuery(bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).WithPageSize(1000)))
		require.NoError(t, err)
		countBefore := len(accountsBefore.Data)

		// Create an account with an invalid connector ID that doesn't exist
		invalidConnectorID := models.ConnectorID{Reference: uuid.New(), Provider: "non-existent-provider"}
		invalidAccount := models.Account{
			ID:          models.AccountID{Reference: "invalid-ref", ConnectorID: invalidConnectorID},
			ConnectorID: invalidConnectorID,
			CreatedAt:   time.Now().UTC().Time,
			Reference:   "invalid-ref",
			Type:        models.ACCOUNT_TYPE_EXTERNAL,
		}

		// Attempt to upsert - should fail due to foreign key violation
		err = store.AccountsUpsert(ctx, []models.Account{invalidAccount})
		require.Error(t, err)

		// Verify no account was inserted
		accountsAfter, err := store.AccountsList(ctx, NewListAccountsQuery(bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).WithPageSize(1000)))
		require.NoError(t, err)
		assert.Equal(t, countBefore, len(accountsAfter.Data), "no accounts should be inserted on error")

		// Verify no outbox events were created
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)
		for _, event := range pendingEvents {
			assert.NotEqual(t, invalidAccount.ID.String(), event.EntityID, "no outbox event should be created for failed insert")
		}
	})
}

func TestAccountsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

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
	defer store.Close()

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
	defer store.Close()

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

	t.Run("list accounts by id", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", defaultAccounts2()[0].ID.String())),
		)

		cursor, err := store.AccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Equal(t, defaultAccounts2()[0], cursor.Data[0])
	})

	t.Run("list accounts by unknown id", func(t *testing.T) {
		q := NewListAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(AccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", "unknown")),
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
		assert.True(t, errors.Is(err, ErrValidation))
		assert.Regexp(t, "metadata\\[foo\\]", err.Error())
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
