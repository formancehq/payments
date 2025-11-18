package storage

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	defaultBankAccount = models.BankAccount{
		ID:            uuid.New(),
		CreatedAt:     now.Add(-60 * time.Minute).UTC().Time,
		Name:          "test1",
		AccountNumber: pointer.For("12345678"),
		Country:       pointer.For("US"),
		Metadata: map[string]string{
			"foo": "bar",
		},
	}

	bcID2               = uuid.New()
	defaultBankAccount2 = models.BankAccount{
		ID:           bcID2,
		CreatedAt:    now.Add(-30 * time.Minute).UTC().Time,
		Name:         "test2",
		IBAN:         pointer.For("DE89370400440532013000"),
		SwiftBicCode: pointer.For("COBADEFFXXX"),
		Country:      pointer.For("DE"),
		Metadata: map[string]string{
			"foo2": "bar2",
		},
		RelatedAccounts: []models.BankAccountRelatedAccount{
			{
				AccountID: defaultAccounts()[0].ID,
				CreatedAt: now.Add(-30 * time.Minute).UTC().Time,
			},
		},
	}

	// No metadata
	defaultBankAccount3 = models.BankAccount{
		ID:            uuid.New(),
		CreatedAt:     now.Add(-55 * time.Minute).UTC().Time,
		Name:          "test1",
		AccountNumber: pointer.For("12345678"),
		Country:       pointer.For("US"),
	}
)

func upsertBankAccount(t *testing.T, ctx context.Context, storage Storage, bankAccounts models.BankAccount) {
	require.NoError(t, storage.BankAccountsUpsert(ctx, bankAccounts))
}

func TestBankAccountsUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	// Helper to clean up outbox events created during tests
	cleanupOutbox := func() {
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 1000)
		require.NoError(t, err)
		for _, event := range pendingEvents {
			eventSent := models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: event.IdempotencyKey,
					ConnectorID:         event.ConnectorID,
				},
				ConnectorID: event.ConnectorID,
				SentAt:      time.Now().UTC(),
			}
			err = store.OutboxEventsDeleteAndRecordSent(ctx, event.ID, eventSent)
			require.NoError(t, err)
		}
	}
	t.Cleanup(func() {
		cleanupOutbox()
		store.Close()
	})

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertBankAccount(t, ctx, store, defaultBankAccount)
	upsertBankAccount(t, ctx, store, defaultBankAccount2)
	cleanupOutbox() // Clean up outbox events from default data

	t.Run("upsert with same id", func(t *testing.T) {
		ba := models.BankAccount{
			ID:            defaultBankAccount.ID,
			CreatedAt:     now.UTC().Time,
			Name:          "changed",
			AccountNumber: pointer.For("987654321"),
			Country:       pointer.For("CA"),
			Metadata: map[string]string{
				"changed": "changed",
			},
		}

		require.NoError(t, store.BankAccountsUpsert(ctx, ba))

		actual, err := store.BankAccountsGet(ctx, ba.ID, true)
		require.NoError(t, err)
		// Should not update the bank account
		compareBankAccounts(t, defaultBankAccount, *actual)
	})

	t.Run("unknown connector id", func(t *testing.T) {
		ba := models.BankAccount{
			ID:            uuid.New(),
			CreatedAt:     now.UTC().Time,
			Name:          "foo",
			AccountNumber: pointer.For("12345678"),
			Country:       pointer.For("US"),
			Metadata: map[string]string{
				"foo": "bar",
			},
			RelatedAccounts: []models.BankAccountRelatedAccount{
				{
					AccountID: models.AccountID{
						Reference: "unknown",
						ConnectorID: models.ConnectorID{
							Reference: uuid.New(),
							Provider:  "unknown",
						},
					},
					CreatedAt: now.UTC().Time,
				},
			},
		}

		require.Error(t, store.BankAccountsUpsert(ctx, ba))
		b, err := store.BankAccountsGet(ctx, ba.ID, true)
		require.Error(t, err)
		require.Nil(t, b)
	})

	t.Run("outbox events created for new bank accounts", func(t *testing.T) {
		t.Cleanup(cleanupOutbox)

		accounts := defaultAccounts()
		// Create new bank account
		newBankAccount := models.BankAccount{
			ID:            uuid.New(),
			CreatedAt:     now.Add(-3 * time.Minute).UTC().Time,
			Name:          "outbox-test-1",
			AccountNumber: pointer.For("987654321"),
			Country:       pointer.For("FR"),
			Metadata: map[string]string{
				"test": "outbox",
			},
			RelatedAccounts: []models.BankAccountRelatedAccount{
				{
					AccountID: accounts[0].ID,
					CreatedAt: now.Add(-3 * time.Minute).UTC().Time,
				},
			},
		}

		expectedKey := newBankAccount.IdempotencyKey()

		// Insert bank account
		require.NoError(t, store.BankAccountsUpsert(ctx, newBankAccount))

		// Verify outbox event was created
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		// Filter events to only the one we just created
		var ourEvent *models.OutboxEvent
		for _, event := range pendingEvents {
			if event.EventType == events.EventTypeSavedBankAccount && event.IdempotencyKey == expectedKey {
				ourEvent = &event
				break
			}
		}
		require.NotNil(t, ourEvent, "expected 1 outbox event for new bank account")

		// Check event details
		assert.Equal(t, events.EventTypeSavedBankAccount, ourEvent.EventType)
		assert.Equal(t, models.OUTBOX_STATUS_PENDING, ourEvent.Status)
		assert.Nil(t, ourEvent.ConnectorID) // Bank accounts don't have connector ID
		assert.Equal(t, 0, ourEvent.RetryCount)
		assert.Nil(t, ourEvent.Error)
		assert.NotEqual(t, uuid.Nil, ourEvent.ID)
		assert.Equal(t, expectedKey, ourEvent.IdempotencyKey)

		// Verify payload contains bank account data
		var payload map[string]interface{}
		err = json.Unmarshal(ourEvent.Payload, &payload)
		require.NoError(t, err)
		assert.Equal(t, newBankAccount.ID.String(), payload["id"])
		assert.Equal(t, newBankAccount.Name, payload["name"])
		compareObfuscatedString(t, *newBankAccount.AccountNumber, payload["accountNumber"].(string))
		assert.Equal(t, *newBankAccount.Country, payload["country"])
		assert.Contains(t, payload, "metadata")
		assert.Contains(t, payload, "createdAt")
		assert.Contains(t, payload, "relatedAccounts")

		// Verify EntityID matches bank account ID
		assert.Equal(t, newBankAccount.ID.String(), ourEvent.EntityID)

		// Verify related accounts in payload
		relatedAccounts, ok := payload["relatedAccounts"].([]interface{})
		require.True(t, ok)
		require.Len(t, relatedAccounts, 1)
		relatedAccount, ok := relatedAccounts[0].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, accounts[0].ID.String(), relatedAccount["accountID"])
		assert.Contains(t, relatedAccount, "connectorID")
		assert.Contains(t, relatedAccount, "provider")
		assert.Contains(t, relatedAccount, "createdAt")
	})

	t.Run("no outbox events for existing bank accounts", func(t *testing.T) {

		t.Cleanup(cleanupOutbox)

		// Try to upsert existing bank account (should not create outbox event due to ON CONFLICT DO NOTHING)
		require.NoError(t, store.BankAccountsUpsert(ctx, defaultBankAccount))

		// Get count of bank_account.saved events after upsert
		allEventsAfter, err := store.OutboxEventsPollPending(ctx, 1000)
		require.NoError(t, err)

		// Verify no new bank account events were created
		assert.Equal(t, 0, len(allEventsAfter), "upserting existing bank account should not create new outbox event")
	})

	t.Run("outbox events created for bank account with all fields", func(t *testing.T) {
		accounts := defaultAccounts()
		// Create bank account with all optional fields
		fullBankAccount := models.BankAccount{
			ID:            uuid.New(),
			CreatedAt:     now.Add(-2 * time.Minute).UTC().Time,
			Name:          "full-test",
			AccountNumber: pointer.For("11111111"),
			IBAN:          pointer.For("GB82WEST12345698765432"),
			SwiftBicCode:  pointer.For("NWBKGB2LXXX"),
			Country:       pointer.For("GB"),
			Metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			RelatedAccounts: []models.BankAccountRelatedAccount{
				{
					AccountID: accounts[0].ID,
					CreatedAt: now.Add(-2 * time.Minute).UTC().Time,
				},
				{
					AccountID: accounts[1].ID,
					CreatedAt: now.Add(-2 * time.Minute).UTC().Time,
				},
			},
		}

		expectedKey := fullBankAccount.IdempotencyKey()

		// Insert bank account
		require.NoError(t, store.BankAccountsUpsert(ctx, fullBankAccount))

		// Verify outbox event was created
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		// Filter events to only the one we just created
		var ourEvent *models.OutboxEvent
		for _, event := range pendingEvents {
			if event.EventType == events.EventTypeSavedBankAccount && event.IdempotencyKey == expectedKey {
				ourEvent = &event
				break
			}
		}
		require.NotNil(t, ourEvent, "expected 1 outbox event for full bank account")

		// Verify payload contains all fields
		var payload map[string]interface{}
		err = json.Unmarshal(ourEvent.Payload, &payload)
		require.NoError(t, err)
		assert.Equal(t, fullBankAccount.ID.String(), payload["id"])
		assert.Equal(t, fullBankAccount.Name, payload["name"])
		compareObfuscatedString(t, *fullBankAccount.AccountNumber, payload["accountNumber"].(string))
		compareObfuscatedString(t, *fullBankAccount.IBAN, payload["iban"].(string))
		assert.Equal(t, *fullBankAccount.SwiftBicCode, payload["swiftBicCode"])
		assert.Equal(t, *fullBankAccount.Country, payload["country"])

		// Verify metadata (JSON unmarshals map[string]string as map[string]interface{})
		metadata, ok := payload["metadata"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, len(fullBankAccount.Metadata), len(metadata))
		for k, v := range fullBankAccount.Metadata {
			assert.Equal(t, v, metadata[k])
		}

		// Verify related accounts
		relatedAccounts, ok := payload["relatedAccounts"].([]interface{})
		require.True(t, ok)
		require.Len(t, relatedAccounts, 2)
	})
}

func TestBankAccountsUpdateMetadata(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertBankAccount(t, ctx, store, defaultBankAccount)
	upsertBankAccount(t, ctx, store, defaultBankAccount2)
	upsertBankAccount(t, ctx, store, defaultBankAccount3)

	t.Run("update metadata", func(t *testing.T) {
		metadata := map[string]string{
			"test1": "test2",
			"test3": "test4",
		}

		// redeclare it in order to not update the map of global variable
		acc := models.BankAccount{
			ID:            defaultBankAccount.ID,
			CreatedAt:     now.Add(-60 * time.Minute).UTC().Time,
			Name:          "test1",
			AccountNumber: pointer.For("12345678"),
			Country:       pointer.For("US"),
			Metadata: map[string]string{
				"foo": "bar",
			},
		}
		for k, v := range metadata {
			acc.Metadata[k] = v
		}

		require.NoError(t, store.BankAccountsUpdateMetadata(ctx, defaultBankAccount.ID, metadata))

		actual, err := store.BankAccountsGet(ctx, defaultBankAccount.ID, true)
		require.NoError(t, err)
		compareBankAccounts(t, acc, *actual)
	})

	t.Run("update same metadata", func(t *testing.T) {
		metadata := map[string]string{
			"foo2": "bar3",
		}

		acc := models.BankAccount{
			ID:           bcID2,
			CreatedAt:    now.Add(-30 * time.Minute).UTC().Time,
			Name:         "test2",
			IBAN:         pointer.For("DE89370400440532013000"),
			SwiftBicCode: pointer.For("COBADEFFXXX"),
			Country:      pointer.For("DE"),
			Metadata: map[string]string{
				"foo2": "bar2",
			},
			RelatedAccounts: []models.BankAccountRelatedAccount{
				{
					AccountID: defaultAccounts()[0].ID,
					CreatedAt: now.Add(-30 * time.Minute).UTC().Time,
				},
			},
		}
		for k, v := range metadata {
			acc.Metadata[k] = v
		}

		require.NoError(t, store.BankAccountsUpdateMetadata(ctx, defaultBankAccount2.ID, metadata))

		actual, err := store.BankAccountsGet(ctx, defaultBankAccount2.ID, true)
		require.NoError(t, err)
		compareBankAccounts(t, acc, *actual)
	})

	t.Run("update metadata of bank accounts with nil map", func(t *testing.T) {
		metadata := map[string]string{
			"test1": "test2",
			"test3": "test4",
		}

		// redeclare it in order to not update the map of global variable
		acc := models.BankAccount{
			ID:            defaultBankAccount3.ID,
			CreatedAt:     now.Add(-55 * time.Minute).UTC().Time,
			Name:          "test1",
			AccountNumber: pointer.For("12345678"),
			Country:       pointer.For("US"),
		}
		acc.Metadata = make(map[string]string)
		for k, v := range metadata {
			acc.Metadata[k] = v
		}

		require.NoError(t, store.BankAccountsUpdateMetadata(ctx, defaultBankAccount3.ID, metadata))

		actual, err := store.BankAccountsGet(ctx, defaultBankAccount3.ID, true)
		require.NoError(t, err)
		compareBankAccounts(t, acc, *actual)
	})
}

func TestBankAccountsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertBankAccount(t, ctx, store, defaultBankAccount)
	upsertBankAccount(t, ctx, store, defaultBankAccount2)
	upsertBankAccount(t, ctx, store, defaultBankAccount3)

	t.Run("get bank account without related accounts", func(t *testing.T) {
		actual, err := store.BankAccountsGet(ctx, defaultBankAccount.ID, true)
		require.NoError(t, err)
		compareBankAccounts(t, defaultBankAccount, *actual)
	})

	t.Run("get bank account without metadata", func(t *testing.T) {
		actual, err := store.BankAccountsGet(ctx, defaultBankAccount3.ID, true)
		require.NoError(t, err)
		compareBankAccounts(t, defaultBankAccount3, *actual)
	})

	t.Run("get bank account with related accounts", func(t *testing.T) {
		actual, err := store.BankAccountsGet(ctx, defaultBankAccount2.ID, true)
		require.NoError(t, err)
		compareBankAccounts(t, defaultBankAccount2, *actual)
	})

	t.Run("get unknown bank account", func(t *testing.T) {
		actual, err := store.BankAccountsGet(ctx, uuid.New(), true)
		require.Error(t, err)
		require.Nil(t, actual)
	})

	t.Run("get bank account with expand to false", func(t *testing.T) {
		acc := models.BankAccount{
			ID:        defaultBankAccount.ID,
			CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
			Name:      "test1",
			Country:   pointer.For("US"),
			Metadata: map[string]string{
				"foo": "bar",
			},
		}

		actual, err := store.BankAccountsGet(ctx, defaultBankAccount.ID, false)
		require.NoError(t, err)
		compareBankAccounts(t, acc, *actual)
	})

	t.Run("get bank account with expand to false 2", func(t *testing.T) {
		acc := models.BankAccount{
			ID:        bcID2,
			CreatedAt: now.Add(-30 * time.Minute).UTC().Time,
			Name:      "test2",
			Country:   pointer.For("DE"),
			Metadata: map[string]string{
				"foo2": "bar2",
			},
			RelatedAccounts: []models.BankAccountRelatedAccount{
				{
					AccountID: defaultAccounts()[0].ID,
					CreatedAt: now.Add(-30 * time.Minute).UTC().Time,
				},
			},
		}

		actual, err := store.BankAccountsGet(ctx, bcID2, false)
		require.NoError(t, err)
		compareBankAccounts(t, acc, *actual)
	})
}

func TestBankAccountsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	d1 := models.BankAccount{
		ID:        defaultBankAccount.ID,
		CreatedAt: defaultBankAccount.CreatedAt,
		Name:      defaultBankAccount.Name,
		Country:   defaultBankAccount.Country,
		Metadata:  defaultBankAccount.Metadata,
	}

	d2 := models.BankAccount{
		ID:              defaultBankAccount2.ID,
		CreatedAt:       defaultBankAccount2.CreatedAt,
		Name:            defaultBankAccount2.Name,
		Country:         defaultBankAccount2.Country,
		Metadata:        defaultBankAccount2.Metadata,
		RelatedAccounts: defaultBankAccount2.RelatedAccounts,
	}
	_ = d2

	d3 := models.BankAccount{
		ID:        defaultBankAccount3.ID,
		CreatedAt: defaultBankAccount3.CreatedAt,
		Name:      defaultBankAccount3.Name,
		Country:   defaultBankAccount3.Country,
	}

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertBankAccount(t, ctx, store, defaultBankAccount)
	upsertBankAccount(t, ctx, store, defaultBankAccount2)
	upsertBankAccount(t, ctx, store, defaultBankAccount3)

	t.Run("wrong query builder operator when listing by name", func(t *testing.T) {
		q := NewListBankAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(BankAccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("name", "test1")),
		)

		cursor, err := store.BankAccountsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
		assert.True(t, errors.Is(err, ErrValidation))
		assert.Regexp(t, "name", err.Error())
	})

	t.Run("list bank accounts by name", func(t *testing.T) {
		q := NewListBankAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(BankAccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("name", "test1")),
		)

		cursor, err := store.BankAccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		compareBankAccounts(t, d3, cursor.Data[0])
		compareBankAccounts(t, d1, cursor.Data[1])
	})

	t.Run("list bank accounts by name 2", func(t *testing.T) {
		q := NewListBankAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(BankAccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("name", "test2")),
		)

		cursor, err := store.BankAccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		compareBankAccounts(t, d2, cursor.Data[0])
	})

	t.Run("list bank accounts by unknown name", func(t *testing.T) {
		q := NewListBankAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(BankAccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("name", "unknown")),
		)

		cursor, err := store.BankAccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
	})

	t.Run("list bank accounts by id", func(t *testing.T) {
		q := NewListBankAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(BankAccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", d3.ID.String())),
		)

		cursor, err := store.BankAccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		compareBankAccounts(t, d3, cursor.Data[0])
	})

	t.Run("list bank accounts by unknown id", func(t *testing.T) {
		q := NewListBankAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(BankAccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", uuid.New().String())),
		)

		cursor, err := store.BankAccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
	})

	t.Run("list bank accounts by country", func(t *testing.T) {
		q := NewListBankAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(BankAccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("country", "US")),
		)

		cursor, err := store.BankAccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		compareBankAccounts(t, d3, cursor.Data[0])
		compareBankAccounts(t, d1, cursor.Data[1])
	})

	t.Run("list bank accounts by country 2", func(t *testing.T) {
		q := NewListBankAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(BankAccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("country", "DE")),
		)

		cursor, err := store.BankAccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		compareBankAccounts(t, d2, cursor.Data[0])
	})

	t.Run("list bank accounts by unknown country", func(t *testing.T) {
		q := NewListBankAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(BankAccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("country", "unknown")),
		)

		cursor, err := store.BankAccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
	})

	t.Run("wrong query builder when listing by metadata", func(t *testing.T) {
		q := NewListBankAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(BankAccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("metadata[foo]", "bar")),
		)

		cursor, err := store.BankAccountsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list bank accounts by metadata", func(t *testing.T) {
		q := NewListBankAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(BankAccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[foo]", "bar")),
		)

		cursor, err := store.BankAccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		compareBankAccounts(t, d1, cursor.Data[0])
	})

	t.Run("list bank accounts by unknown metadata", func(t *testing.T) {
		q := NewListBankAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(BankAccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[unknown]", "bar")),
		)

		cursor, err := store.BankAccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
	})

	t.Run("unknown query builder key when listing bank accounts", func(t *testing.T) {
		q := NewListBankAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(BankAccountQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("unknown", "bar")),
		)

		cursor, err := store.BankAccountsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list bank accounts test cursor", func(t *testing.T) {
		q := NewListBankAccountsQuery(
			bunpaginate.NewPaginatedQueryOptions(BankAccountQuery{}).
				WithPageSize(1),
		)

		cursor, err := store.BankAccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		compareBankAccounts(t, d2, cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.BankAccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		compareBankAccounts(t, d3, cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.BankAccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		compareBankAccounts(t, d1, cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.BankAccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		compareBankAccounts(t, d3, cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.BankAccountsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		compareBankAccounts(t, d2, cursor.Data[0])
	})
}

func TestBankAccountsAddRelatedAccount(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertBankAccount(t, ctx, store, defaultBankAccount)
	upsertBankAccount(t, ctx, store, defaultBankAccount2)
	upsertBankAccount(t, ctx, store, defaultBankAccount3)

	t.Run("add related account when empty", func(t *testing.T) {
		acc := models.BankAccountRelatedAccount{
			AccountID: defaultAccounts()[0].ID,
			CreatedAt: now.UTC().Time,
		}

		ba := defaultBankAccount
		ba.RelatedAccounts = append(ba.RelatedAccounts, acc)

		require.NoError(t, store.BankAccountsAddRelatedAccount(ctx, ba.ID, acc))

		actual, err := store.BankAccountsGet(ctx, defaultBankAccount.ID, true)
		require.NoError(t, err)
		compareBankAccounts(t, ba, *actual)
	})

	t.Run("add related account when not empty", func(t *testing.T) {
		acc := models.BankAccountRelatedAccount{
			AccountID: defaultAccounts()[1].ID,
			CreatedAt: now.UTC().Time,
		}

		ba := defaultBankAccount2
		ba.RelatedAccounts = append(ba.RelatedAccounts, acc)

		require.NoError(t, store.BankAccountsAddRelatedAccount(ctx, defaultBankAccount2.ID, acc))

		actual, err := store.BankAccountsGet(ctx, defaultBankAccount2.ID, true)
		require.NoError(t, err)
		compareBankAccounts(t, ba, *actual)
	})

	t.Run("add related account with unknown bank account", func(t *testing.T) {
		acc := models.BankAccountRelatedAccount{
			AccountID: defaultAccounts()[1].ID,
			CreatedAt: now.UTC().Time,
		}

		require.Error(t, store.BankAccountsAddRelatedAccount(ctx, uuid.New(), acc))
	})

	t.Run("add related account with unknown connector", func(t *testing.T) {
		acc := models.BankAccountRelatedAccount{
			AccountID: models.AccountID{
				Reference: "unknown",
				ConnectorID: models.ConnectorID{
					Reference: uuid.New(),
					Provider:  "unknown",
				},
			},
			CreatedAt: now.UTC().Time,
		}

		require.Error(t, store.BankAccountsAddRelatedAccount(ctx, defaultBankAccount.ID, acc))
	})

	t.Run("add related account with existing related account", func(t *testing.T) {
		acc := models.BankAccountRelatedAccount{
			AccountID: defaultAccounts()[0].ID,
			CreatedAt: now.Add(-30 * time.Minute).UTC().Time,
		}

		ba := defaultBankAccount3
		ba.RelatedAccounts = append(ba.RelatedAccounts, acc)

		require.NoError(t, store.BankAccountsAddRelatedAccount(ctx, defaultBankAccount3.ID, acc))

		actual, err := store.BankAccountsGet(ctx, defaultBankAccount3.ID, true)
		require.NoError(t, err)
		compareBankAccounts(t, ba, *actual)

		require.NoError(t, store.BankAccountsAddRelatedAccount(ctx, defaultBankAccount3.ID, acc))

		actual, err = store.BankAccountsGet(ctx, defaultBankAccount3.ID, true)
		require.NoError(t, err)
		compareBankAccounts(t, ba, *actual)
	})
}

func TestBankAccountsDeleteRelatedAccountFromConnectorID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertBankAccount(t, ctx, store, defaultBankAccount)
	upsertBankAccount(t, ctx, store, defaultBankAccount2)
	upsertBankAccount(t, ctx, store, defaultBankAccount3)

	t.Run("delete related account with unknown connector", func(t *testing.T) {
		require.NoError(t, store.BankAccountsDeleteRelatedAccountFromConnectorID(ctx, models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}))

		actual, err := store.BankAccountsGet(ctx, defaultBankAccount2.ID, true)
		require.NoError(t, err)
		compareBankAccounts(t, defaultBankAccount2, *actual)
	})

	t.Run("delete related account with another connector id", func(t *testing.T) {
		require.NoError(t, store.BankAccountsDeleteRelatedAccountFromConnectorID(ctx, defaultConnector2.ID))

		actual, err := store.BankAccountsGet(ctx, defaultBankAccount2.ID, true)
		require.NoError(t, err)
		compareBankAccounts(t, defaultBankAccount2, *actual)
	})

	t.Run("delete related account", func(t *testing.T) {
		require.NoError(t, store.BankAccountsDeleteRelatedAccountFromConnectorID(ctx, defaultConnector.ID))

		ba := defaultBankAccount2
		ba.RelatedAccounts = nil

		actual, err := store.BankAccountsGet(ctx, defaultBankAccount2.ID, true)
		require.NoError(t, err)
		compareBankAccounts(t, ba, *actual)
	})
}

func compareBankAccounts(t *testing.T, expected, actual models.BankAccount) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Equal(t, expected.Name, actual.Name)

	require.Equal(t, len(expected.Metadata), len(actual.Metadata))
	for k, v := range expected.Metadata {
		require.Equal(t, v, actual.Metadata[k])
	}
	for k, v := range actual.Metadata {
		require.Equal(t, v, expected.Metadata[k])
	}

	switch {
	case expected.AccountNumber != nil && actual.AccountNumber != nil:
		require.Equal(t, *expected.AccountNumber, *actual.AccountNumber)
	case expected.AccountNumber == nil && actual.AccountNumber == nil:
		// Nothing to do
	default:
		require.Fail(t, "AccountNumber mismatch")
	}

	switch {
	case expected.IBAN != nil && actual.IBAN != nil:
		require.Equal(t, *expected.IBAN, *actual.IBAN)
	case expected.IBAN == nil && actual.IBAN == nil:
		// Nothing to do
	default:
		require.Fail(t, "IBAN mismatch")
	}

	switch {
	case expected.SwiftBicCode != nil && actual.SwiftBicCode != nil:
		require.Equal(t, *expected.SwiftBicCode, *actual.SwiftBicCode)
	case expected.SwiftBicCode == nil && actual.SwiftBicCode == nil:
		// Nothing to do
	default:
		require.Fail(t, "SwiftBicCode mismatch")
	}

	switch {
	case expected.Country != nil && actual.Country != nil:
		require.Equal(t, *expected.Country, *actual.Country)
	case expected.Country == nil && actual.Country == nil:
		// Nothing to do
	default:
		require.Fail(t, "Country mismatch")
	}

	require.Equal(t, len(expected.RelatedAccounts), len(actual.RelatedAccounts))
	for i := range expected.RelatedAccounts {
		require.Equal(t, expected.RelatedAccounts[i], actual.RelatedAccounts[i])
	}
}

func compareObfuscatedString(t *testing.T, expected, actual string) {
	if expected == "" {
		assert.Empty(t, actual)
		return
	}
	assert.True(t, len(expected) > 2)
	assert.True(t, len(actual) > 2)

	assert.True(t, len(actual) == len(expected))
	assert.NotEqual(t, expected, actual)

	assert.Equal(t, expected[0:2], actual[0:2])
	assert.Equal(t, expected[len(expected)-2:], actual[len(actual)-2:])
}
