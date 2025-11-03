package storage

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultBalances() []models.Balance {
	defaultAccounts := defaultAccounts()
	return []models.Balance{
		{
			AccountID:     defaultAccounts[0].ID,
			CreatedAt:     now.Add(-60 * time.Minute).UTC().Time,
			LastUpdatedAt: now.Add(-60 * time.Minute).UTC().Time,
			Asset:         "USD/2",
			Balance:       big.NewInt(100),
		},
		{
			AccountID:     defaultAccounts[1].ID,
			CreatedAt:     now.Add(-30 * time.Minute).UTC().Time,
			LastUpdatedAt: now.Add(-30 * time.Minute).UTC().Time,
			Asset:         "EUR/2",
			Balance:       big.NewInt(1000),
		},
		{
			AccountID:               defaultAccounts[0].ID,
			CreatedAt:               now.Add(-55 * time.Minute).UTC().Time,
			LastUpdatedAt:           now.Add(-55 * time.Minute).UTC().Time,
			Asset:                   "EUR/2",
			Balance:                 big.NewInt(150),
			PsuID:                   &defaultPSU2.ID,
			OpenBankingConnectionID: &defaultOpenBankingConnection.ConnectionID,
		},
	}
}

func defaultBalances2() []models.Balance {
	defaultAccounts := defaultAccounts()
	return []models.Balance{
		{
			AccountID:     defaultAccounts[2].ID,
			CreatedAt:     now.Add(-59 * time.Minute).UTC().Time,
			LastUpdatedAt: now.Add(-59 * time.Minute).UTC().Time,
			Asset:         "USD/2",
			Balance:       big.NewInt(100),
		},
		{
			AccountID:     defaultAccounts[2].ID,
			CreatedAt:     now.Add(-31 * time.Minute).UTC().Time,
			LastUpdatedAt: now.Add(-31 * time.Minute).UTC().Time,
			Asset:         "DKK/2",
			Balance:       big.NewInt(1000),
		},
	}
}

func upsertBalances(t *testing.T, ctx context.Context, storage Storage, balances []models.Balance) {
	require.NoError(t, storage.BalancesUpsert(ctx, balances))
}

func TestBalancesUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

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
			_ = store.OutboxEventsDeleteAndRecordSent(ctx, event.ID, eventSent)
		}
	}
	defer cleanupOutbox()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultOpenBankingConnection)
	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertBalances(t, ctx, store, defaultBalances())
	upsertBalances(t, ctx, store, defaultBalances2())
	cleanupOutbox() // Clean up outbox events from default data creation

	t.Run("upsert empty balances", func(t *testing.T) {
		upsertBalances(t, ctx, store, []models.Balance{})
	})

	t.Run("upsert balances with unknown connector id", func(t *testing.T) {
		b := models.Balance{
			AccountID: models.AccountID{
				Reference: "test",
				ConnectorID: models.ConnectorID{
					Reference: uuid.New(),
					Provider:  "unknown",
				},
			},
			CreatedAt:     now.Add(-70 * time.Minute).UTC().Time,
			LastUpdatedAt: now.Add(-70 * time.Minute).UTC().Time,
			Asset:         "USD/2",
			Balance:       big.NewInt(100),
		}

		require.Error(t, store.BalancesUpsert(ctx, []models.Balance{b}))
	})

	t.Run("upsert balance in the past should not insert anything", func(t *testing.T) {
		accounts := defaultAccounts()
		b := models.Balance{
			AccountID:     accounts[0].ID,
			CreatedAt:     now.Add(-70 * time.Minute).UTC().Time,
			LastUpdatedAt: now.Add(-70 * time.Minute).UTC().Time,
			Asset:         "USD/2",
			Balance:       big.NewInt(100),
		}

		upsertBalances(t, ctx, store, []models.Balance{b})

		q := NewListBalancesQuery(
			bunpaginate.NewPaginatedQueryOptions(BalanceQuery{
				AccountID: pointer.For(accounts[0].ID),
				Asset:     "USD/2",
			}).WithPageSize(15),
		)

		// We should have the same balances as before
		expectedBalances := []models.Balance{
			{
				AccountID:     accounts[0].ID,
				CreatedAt:     now.Add(-60 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-60 * time.Minute).UTC().Time,
				Asset:         "USD/2",
				Balance:       big.NewInt(100),
			},
		}

		balances, err := store.BalancesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, balances.Data, 1)
		require.Equal(t, expectedBalances, balances.Data)
	})

	t.Run("insert balances with same asset and same balance", func(t *testing.T) {
		accounts := defaultAccounts()
		b := models.Balance{
			AccountID:     accounts[2].ID,
			CreatedAt:     now.Add(-20 * time.Minute).UTC().Time,
			LastUpdatedAt: now.Add(-20 * time.Minute).UTC().Time,
			Asset:         "USD/2",
			Balance:       big.NewInt(100),
		}

		upsertBalances(t, ctx, store, []models.Balance{b})

		q := NewListBalancesQuery(
			bunpaginate.NewPaginatedQueryOptions(BalanceQuery{
				AccountID: pointer.For(accounts[2].ID),
				Asset:     "USD/2",
			}).WithPageSize(15),
		)

		expectedBalances := []models.Balance{
			{
				AccountID:     accounts[2].ID,
				CreatedAt:     now.Add(-59 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-20 * time.Minute).UTC().Time, // Last updated at should be updated to the new balance value
				Asset:         "USD/2",
				Balance:       big.NewInt(100),
			},
		}

		balances, err := store.BalancesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, balances.Data, 1)
		require.Equal(t, expectedBalances, balances.Data)
	})

	t.Run("insert balances same asset different balance", func(t *testing.T) {
		accounts := defaultAccounts()
		b := models.Balance{
			AccountID:     accounts[0].ID,
			CreatedAt:     now.Add(-20 * time.Minute).UTC().Time,
			LastUpdatedAt: now.Add(-20 * time.Minute).UTC().Time,
			Asset:         "USD/2",
			Balance:       big.NewInt(200),
		}

		upsertBalances(t, ctx, store, []models.Balance{b})

		q := NewListBalancesQuery(
			bunpaginate.NewPaginatedQueryOptions(BalanceQuery{
				AccountID: pointer.For(accounts[0].ID),
				Asset:     "USD/2",
			}).WithPageSize(15),
		)

		expectedBalances := []models.Balance{
			// We should have one more balance with the new balance value
			{
				AccountID:     accounts[0].ID,
				CreatedAt:     now.Add(-20 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-20 * time.Minute).UTC().Time,
				Asset:         "USD/2",
				Balance:       big.NewInt(200),
			},
			// and the old balance should have its updated at to the new balance created at
			{
				AccountID:     accounts[0].ID,
				CreatedAt:     now.Add(-60 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-20 * time.Minute).UTC().Time,
				Asset:         "USD/2",
				Balance:       big.NewInt(100),
			},
		}

		balances, err := store.BalancesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, balances.Data, 2)
		require.Equal(t, expectedBalances, balances.Data)
	})

	t.Run("insert balances with new asset", func(t *testing.T) {
		accounts := defaultAccounts()
		b := models.Balance{
			AccountID:     accounts[2].ID,
			CreatedAt:     now.Add(-10 * time.Minute).UTC().Time,
			LastUpdatedAt: now.Add(-10 * time.Minute).UTC().Time,
			Asset:         "EUR/2",
			Balance:       big.NewInt(200),
		}

		upsertBalances(t, ctx, store, []models.Balance{b})

		q := NewListBalancesQuery(
			bunpaginate.NewPaginatedQueryOptions(BalanceQuery{
				AccountID: pointer.For(accounts[2].ID),
			}).WithPageSize(15),
		)

		expectedBalances := []models.Balance{
			{
				AccountID:     accounts[2].ID,
				CreatedAt:     now.Add(-10 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-10 * time.Minute).UTC().Time,
				Asset:         "EUR/2",
				Balance:       big.NewInt(200),
			},
			{
				AccountID:     accounts[2].ID,
				CreatedAt:     now.Add(-31 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-31 * time.Minute).UTC().Time,
				Asset:         "DKK/2",
				Balance:       big.NewInt(1000),
			},
			{
				AccountID:     accounts[2].ID,
				CreatedAt:     now.Add(-59 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-20 * time.Minute).UTC().Time, // Because on the first function it was modified
				Asset:         "USD/2",
				Balance:       big.NewInt(100),
			},
		}

		cursor, err := store.BalancesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 3)
		require.Equal(t, expectedBalances, cursor.Data)
	})

	t.Run("no balances available", func(t *testing.T) {
		accountID := models.AccountID{
			Reference:   "12324343abc",
			ConnectorID: defaultConnector.ID,
		}

		q := NewListBalancesQuery(
			bunpaginate.NewPaginatedQueryOptions(BalanceQuery{
				AccountID: pointer.For(accountID),
			}).WithPageSize(15),
		)

		expectedBalances := []models.Balance{}
		cursor, err := store.BalancesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.Equal(t, expectedBalances, cursor.Data)
	})

	t.Run("outbox events created for new balances", func(t *testing.T) {
		accounts := defaultAccounts()
		// Create new balances
		newBalances := []models.Balance{
			{
				AccountID:     accounts[0].ID,
				CreatedAt:     now.Add(-5 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-5 * time.Minute).UTC().Time,
				Asset:         "GBP/2",
				Balance:       big.NewInt(500),
			},
			{
				AccountID:     accounts[1].ID,
				CreatedAt:     now.Add(-4 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-4 * time.Minute).UTC().Time,
				Asset:         "JPY/0",
				Balance:       big.NewInt(10000),
			},
		}

		// Create a set of expected idempotency keys
		expectedKeys := make(map[string]bool)
		for _, balance := range newBalances {
			expectedKeys[balance.IdempotencyKey()] = true
		}

		// Insert balances
		require.NoError(t, store.BalancesUpsert(ctx, newBalances))

		// Verify outbox events were created
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		// Filter events to only those we just created
		ourEvents := make([]models.OutboxEvent, 0)
		for _, event := range pendingEvents {
			if event.EventType == "balance.saved" && expectedKeys[event.IdempotencyKey] {
				ourEvents = append(ourEvents, event)
			}
		}
		require.Len(t, ourEvents, 2, "expected 2 outbox events for 2 new balances")

		// Create a map of expected balances by idempotency key for easier lookup
		expectedBalancesByKey := make(map[string]models.Balance)
		for _, balance := range newBalances {
			expectedBalancesByKey[balance.IdempotencyKey()] = balance
		}

		// Check event details
		for _, event := range ourEvents {
			assert.Equal(t, "balance.saved", event.EventType)
			assert.Equal(t, models.OUTBOX_STATUS_PENDING, event.Status)
			assert.Equal(t, defaultConnector.ID, *event.ConnectorID)
			assert.Equal(t, 0, event.RetryCount)
			assert.Nil(t, event.Error)
			assert.NotEqual(t, uuid.Nil, event.ID)
			assert.NotEmpty(t, event.IdempotencyKey)

			// Find the matching balance by idempotency key
			expectedBalance, found := expectedBalancesByKey[event.IdempotencyKey]
			require.True(t, found, "event idempotency key should match one of the balances")

			// Verify payload contains balance data
			var payload map[string]interface{}
			err := json.Unmarshal(event.Payload, &payload)
			require.NoError(t, err)
			assert.Equal(t, expectedBalance.AccountID.String(), payload["accountID"])
			assert.Equal(t, expectedBalance.AccountID.ConnectorID.String(), payload["connectorID"])
			assert.Equal(t, expectedBalance.Asset, payload["asset"])
			assert.Equal(t, expectedBalance.Balance.String(), payload["balance"])
			assert.Contains(t, payload, "provider")
			assert.Contains(t, payload, "createdAt")
			assert.Contains(t, payload, "lastUpdatedAt")

			// Verify EntityID matches account ID
			assert.Equal(t, expectedBalance.AccountID.String(), event.EntityID)
		}
	})

	t.Run("outbox events created for balance updates with different values", func(t *testing.T) {
		accounts := defaultAccounts()
		// Update a balance with a different value (should create a new entry)
		updatedBalance := models.Balance{
			AccountID:     accounts[0].ID,
			CreatedAt:     now.Add(-2 * time.Minute).UTC().Time,
			LastUpdatedAt: now.Add(-2 * time.Minute).UTC().Time,
			Asset:         "USD/2",
			Balance:       big.NewInt(300), // Different from the existing balance
		}

		expectedKey := updatedBalance.IdempotencyKey()

		// Insert balance update
		require.NoError(t, store.BalancesUpsert(ctx, []models.Balance{updatedBalance}))

		// Verify outbox event was created
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		// Filter events to only the one we just created
		var ourEvent *models.OutboxEvent
		for _, event := range pendingEvents {
			if event.EventType == "balance.saved" && event.IdempotencyKey == expectedKey {
				ourEvent = &event
				break
			}
		}
		require.NotNil(t, ourEvent, "expected 1 outbox event for balance update")

		assert.Equal(t, "balance.saved", ourEvent.EventType)
		assert.Equal(t, models.OUTBOX_STATUS_PENDING, ourEvent.Status)

		// Verify payload
		var payload map[string]interface{}
		err = json.Unmarshal(ourEvent.Payload, &payload)
		require.NoError(t, err)
		assert.Equal(t, updatedBalance.Balance.String(), payload["balance"])
	})

	t.Run("outbox events created for multiple balances in single upsert", func(t *testing.T) {
		accounts := defaultAccounts()
		// Create multiple balances at once
		multiBalances := []models.Balance{
			{
				AccountID:     accounts[2].ID,
				CreatedAt:     now.Add(-1 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-1 * time.Minute).UTC().Time,
				Asset:         "CHF/2",
				Balance:       big.NewInt(150),
			},
			{
				AccountID:     accounts[2].ID,
				CreatedAt:     now.Add(-1 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-1 * time.Minute).UTC().Time,
				Asset:         "AUD/2",
				Balance:       big.NewInt(200),
			},
		}

		// Create a set of expected idempotency keys
		expectedKeys := make(map[string]bool)
		for _, balance := range multiBalances {
			expectedKeys[balance.IdempotencyKey()] = true
		}

		// Insert multiple balances
		require.NoError(t, store.BalancesUpsert(ctx, multiBalances))

		// Verify outbox events were created for all balances
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		// Filter events to only those we just created
		ourEvents := make([]models.OutboxEvent, 0)
		for _, event := range pendingEvents {
			if event.EventType == "balance.saved" && expectedKeys[event.IdempotencyKey] {
				ourEvents = append(ourEvents, event)
			}
		}
		require.Len(t, ourEvents, 2, "expected 2 outbox events for 2 balances")

		// Verify all events have correct structure
		for _, event := range ourEvents {
			assert.Equal(t, "balance.saved", event.EventType)
			assert.Equal(t, models.OUTBOX_STATUS_PENDING, event.Status)
			assert.NotEqual(t, uuid.Nil, event.ID)
			assert.NotEmpty(t, event.IdempotencyKey)
		}
	})
}

func TestBalancesList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultOpenBankingConnection)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertBalances(t, ctx, store, defaultBalances())
	upsertBalances(t, ctx, store, defaultBalances2())

	t.Run("list balances with account id", func(t *testing.T) {
		accounts := defaultAccounts()
		q := NewListBalancesQuery(
			bunpaginate.NewPaginatedQueryOptions(BalanceQuery{
				AccountID: pointer.For(accounts[0].ID),
			}).WithPageSize(15),
		)

		expectedBalances := []models.Balance{
			{
				AccountID:               accounts[0].ID,
				CreatedAt:               now.Add(-55 * time.Minute).UTC().Time,
				LastUpdatedAt:           now.Add(-55 * time.Minute).UTC().Time,
				Asset:                   "EUR/2",
				Balance:                 big.NewInt(150),
				PsuID:                   &defaultPSU2.ID,
				OpenBankingConnectionID: &defaultOpenBankingConnection.ConnectionID,
			},
			{
				AccountID:               accounts[0].ID,
				CreatedAt:               now.Add(-60 * time.Minute).UTC().Time,
				LastUpdatedAt:           now.Add(-60 * time.Minute).UTC().Time,
				Asset:                   "USD/2",
				Balance:                 big.NewInt(100),
				PsuID:                   nil,
				OpenBankingConnectionID: nil,
			},
		}

		cursor, err := store.BalancesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		require.Equal(t, expectedBalances, cursor.Data)
	})

	t.Run("list balances with asset 1", func(t *testing.T) {
		q := NewListBalancesQuery(
			bunpaginate.NewPaginatedQueryOptions(BalanceQuery{
				Asset: "USD/2",
			}).WithPageSize(15),
		)

		accounts := defaultAccounts()
		expectedBalances := []models.Balance{
			{
				AccountID:     accounts[2].ID,
				CreatedAt:     now.Add(-59 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-59 * time.Minute).UTC().Time,
				Asset:         "USD/2",
				Balance:       big.NewInt(100),
			},
			{
				AccountID:     accounts[0].ID,
				CreatedAt:     now.Add(-60 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-60 * time.Minute).UTC().Time,
				Asset:         "USD/2",
				Balance:       big.NewInt(100),
			},
		}

		cursor, err := store.BalancesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		require.Equal(t, expectedBalances, cursor.Data)
	})

	t.Run("list balances with asset 2", func(t *testing.T) {
		q := NewListBalancesQuery(
			bunpaginate.NewPaginatedQueryOptions(BalanceQuery{
				Asset: "DKK/2",
			}).WithPageSize(15),
		)

		accounts := defaultAccounts()
		expectedBalances := []models.Balance{
			{
				AccountID:     accounts[2].ID,
				CreatedAt:     now.Add(-31 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-31 * time.Minute).UTC().Time,
				Asset:         "DKK/2",
				Balance:       big.NewInt(1000),
			},
		}

		cursor, err := store.BalancesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Equal(t, expectedBalances, cursor.Data)
	})

	t.Run("list balances with from", func(t *testing.T) {
		q := NewListBalancesQuery(
			bunpaginate.NewPaginatedQueryOptions(NewBalanceQuery().WithFrom(now.Add(-40 * time.Minute).UTC().Time)).WithPageSize(15),
		)

		accounts := defaultAccounts()
		expectedBalances := []models.Balance{
			{
				AccountID:     accounts[1].ID,
				CreatedAt:     now.Add(-30 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-30 * time.Minute).UTC().Time,
				Asset:         "EUR/2",
				Balance:       big.NewInt(1000),
			},
			{
				AccountID:     accounts[2].ID,
				CreatedAt:     now.Add(-31 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-31 * time.Minute).UTC().Time,
				Asset:         "DKK/2",
				Balance:       big.NewInt(1000),
			},
		}

		cursor, err := store.BalancesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		require.Equal(t, expectedBalances, cursor.Data)
	})

	t.Run("list balances with from 2", func(t *testing.T) {
		q := NewListBalancesQuery(
			bunpaginate.NewPaginatedQueryOptions(BalanceQuery{
				From: now.Add(-20 * time.Minute).UTC().Time,
			}).WithPageSize(15),
		)

		cursor, err := store.BalancesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list balances with to", func(t *testing.T) {
		q := NewListBalancesQuery(
			bunpaginate.NewPaginatedQueryOptions(BalanceQuery{
				To: now.Add(-40 * time.Minute).UTC().Time,
			}).WithPageSize(15),
		)

		accounts := defaultAccounts()
		expectedBalances := []models.Balance{
			{
				AccountID:               accounts[0].ID,
				CreatedAt:               now.Add(-55 * time.Minute).UTC().Time,
				LastUpdatedAt:           now.Add(-55 * time.Minute).UTC().Time,
				Asset:                   "EUR/2",
				Balance:                 big.NewInt(150),
				PsuID:                   &defaultPSU2.ID,
				OpenBankingConnectionID: &defaultOpenBankingConnection.ConnectionID,
			},
			{
				AccountID:     accounts[2].ID,
				CreatedAt:     now.Add(-59 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-59 * time.Minute).UTC().Time,
				Asset:         "USD/2",
				Balance:       big.NewInt(100),
			},
			{
				AccountID:     accounts[0].ID,
				CreatedAt:     now.Add(-60 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-60 * time.Minute).UTC().Time,
				Asset:         "USD/2",
				Balance:       big.NewInt(100),
			},
		}

		cursor, err := store.BalancesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 3)
		require.False(t, cursor.HasMore)
		require.Equal(t, expectedBalances, cursor.Data)
	})

	t.Run("list balances with to 2", func(t *testing.T) {
		q := NewListBalancesQuery(
			bunpaginate.NewPaginatedQueryOptions(BalanceQuery{
				To: now.Add(-70 * time.Minute).UTC().Time,
			}).WithPageSize(15),
		)

		cursor, err := store.BalancesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list balances test cursor", func(t *testing.T) {
		accounts := defaultAccounts()
		q := NewListBalancesQuery(
			bunpaginate.NewPaginatedQueryOptions(BalanceQuery{
				AccountID: pointer.For(accounts[0].ID),
			}).WithPageSize(1),
		)
		expectedBalances1 := []models.Balance{
			{
				AccountID:               accounts[0].ID,
				CreatedAt:               now.Add(-55 * time.Minute).UTC().Time,
				LastUpdatedAt:           now.Add(-55 * time.Minute).UTC().Time,
				Asset:                   "EUR/2",
				Balance:                 big.NewInt(150),
				PsuID:                   &defaultPSU2.ID,
				OpenBankingConnectionID: &defaultOpenBankingConnection.ConnectionID,
			},
		}

		cursor, err := store.BalancesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		require.Equal(t, expectedBalances1, cursor.Data)

		expectedBalances2 := []models.Balance{
			{
				AccountID:     accounts[0].ID,
				CreatedAt:     now.Add(-60 * time.Minute).UTC().Time,
				LastUpdatedAt: now.Add(-60 * time.Minute).UTC().Time,
				Asset:         "USD/2",
				Balance:       big.NewInt(100),
			},
		}

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.BalancesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Next)
		require.NotEmpty(t, cursor.Previous)
		require.Equal(t, expectedBalances2, cursor.Data)

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.BalancesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		require.Equal(t, expectedBalances1, cursor.Data)
	})
}

func TestBalancesGetAt(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultOpenBankingConnection)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertBalances(t, ctx, store, defaultBalances())

	t.Run("get balances at before first balance should return empty", func(t *testing.T) {
		accounts := defaultAccounts()
		balances, err := store.BalancesGetAt(ctx, accounts[0].ID, now.Add(-61*time.Minute).UTC().Time)
		require.NoError(t, err)
		require.Nil(t, balances)
	})

	t.Run("get balances at after last balance updated at should return empty", func(t *testing.T) {
		accounts := defaultAccounts()
		balances, err := store.BalancesGetAt(ctx, accounts[0].ID, now.Add(-50*time.Minute).UTC().Time)
		require.NoError(t, err)
		require.Nil(t, balances)
	})

	t.Run("get balances at", func(t *testing.T) {
		accounts := defaultAccounts()
		balances, err := store.BalancesGetAt(ctx, accounts[0].ID, now.Add(-60*time.Minute).UTC().Time)
		require.NoError(t, err)
		require.NotNil(t, balances)
		require.Len(t, balances, 1)
	})

	t.Run("get balances after inserting a new balance", func(t *testing.T) {
		accounts := defaultAccounts()
		b := models.Balance{
			AccountID:     accounts[0].ID,
			CreatedAt:     now.Add(-20 * time.Minute).UTC().Time,
			LastUpdatedAt: now.Add(-20 * time.Minute).UTC().Time,
			Asset:         "USD/2",
			Balance:       big.NewInt(100),
		}

		upsertBalances(t, ctx, store, []models.Balance{b})

		balances, err := store.BalancesGetAt(ctx, accounts[0].ID, now.Add(-50*time.Minute).UTC().Time)
		require.NoError(t, err)
		require.NotNil(t, balances)
		require.Len(t, balances, 1)
	})

	t.Run("get balances at after inserting two new balances with different asset", func(t *testing.T) {
		accounts := defaultAccounts()

		b := models.Balance{
			AccountID:     accounts[0].ID,
			CreatedAt:     now.Add(-20 * time.Minute).UTC().Time,
			LastUpdatedAt: now.Add(-20 * time.Minute).UTC().Time,
			Asset:         "USD/2",
			Balance:       big.NewInt(100),
		}

		b1 := models.Balance{
			AccountID:     accounts[0].ID,
			CreatedAt:     now.Add(-20 * time.Minute).UTC().Time,
			LastUpdatedAt: now.Add(-20 * time.Minute).UTC().Time,
			Asset:         "EUR/2",
			Balance:       big.NewInt(100),
		}

		upsertBalances(t, ctx, store, []models.Balance{b, b1})

		balances, err := store.BalancesGetAt(ctx, accounts[0].ID, now.Add(-50*time.Minute).UTC().Time)
		require.NoError(t, err)
		require.NotNil(t, balances)
		require.Len(t, balances, 2)
	})
}

func TestBalancesGetLatest(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultOpenBankingConnection)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertBalances(t, ctx, store, defaultBalances())

	t.Run("get latest balances returns 1 balance per currency", func(t *testing.T) {
		accounts := defaultAccounts()
		balances, err := store.BalancesGetLatest(ctx, accounts[0].ID)
		require.NoError(t, err)
		require.NotNil(t, balances)
		require.Len(t, balances, 2)
		assert.Equal(t, balances[0].Asset, "EUR/2")
		assert.Equal(t, balances[1].Asset, "USD/2")
	})

	t.Run("get balances after inserting a new balance", func(t *testing.T) {
		accounts := defaultAccounts()
		b := models.Balance{
			AccountID:     accounts[0].ID,
			CreatedAt:     now.Add(-20 * time.Minute).UTC().Time,
			LastUpdatedAt: now.Add(-20 * time.Minute).UTC().Time,
			Asset:         "USD/2",
			Balance:       big.NewInt(999),
		}

		upsertBalances(t, ctx, store, []models.Balance{b})

		balances, err := store.BalancesGetLatest(ctx, accounts[0].ID)
		require.NoError(t, err)
		require.NotNil(t, balances)
		require.Len(t, balances, 2)
		assert.Equal(t, balances[1].Asset, "USD/2")
		assert.Equal(t, balances[1].Balance, b.Balance)
	})
}
