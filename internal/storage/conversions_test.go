package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConversionsUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)

	t.Run("outbox event created atomically with conversion", func(t *testing.T) {
		defer cleanupOutboxHelper(ctx, store)()

		convID := models.ConversionID{
			Reference:   "conv-test-1",
			ConnectorID: defaultConnector.ID,
		}

		conv := models.Conversion{
			ID:               convID,
			ConnectorID:      defaultConnector.ID,
			Reference:        "conv-test-1",
			CreatedAt:        now.Add(-60 * time.Minute).UTC().Time,
			UpdatedAt:        now.Add(-5 * time.Minute).UTC().Time,
			SourceAsset:      "USD/2",
			DestinationAsset: "USDC/6",
			SourceAmount:     big.NewInt(10000),
			DestinationAmount: big.NewInt(10000000),
			Status:           models.CONVERSION_STATUS_COMPLETED,
			Fee:              big.NewInt(50),
			FeeAsset:         pointer.For("USD/2"),
			Metadata:         map[string]string{},
			Raw:              []byte(`{"test": "data"}`),
		}

		require.NoError(t, store.ConversionsUpsert(ctx, []models.Conversion{conv}))

		// Verify conversion exists
		storedConv, err := store.ConversionsGet(ctx, convID)
		require.NoError(t, err)
		assert.Equal(t, convID, storedConv.ID)
		assert.Equal(t, models.CONVERSION_STATUS_COMPLETED, storedConv.Status)

		// Verify outbox event exists
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		expectedKey := conv.IdempotencyKey()
		var found bool
		for _, event := range pendingEvents {
			if event.ID.EventIdempotencyKey == expectedKey {
				found = true
				assert.Equal(t, events.EventTypeSavedConversion, event.EventType)
				assert.Equal(t, convID.String(), event.EntityID)
				assert.Equal(t, models.OUTBOX_STATUS_PENDING, event.Status)
				assert.Equal(t, defaultConnector.ID, *event.ConnectorID)

				// Verify payload
				var payload map[string]interface{}
				err = json.Unmarshal(event.Payload, &payload)
				require.NoError(t, err)
				assert.Equal(t, conv.ID.String(), payload["id"])
				assert.Equal(t, conv.Status.String(), payload["status"])
				break
			}
		}
		assert.True(t, found, "outbox event must exist for new conversion")
	})

	t.Run("duplicate conversion does not create duplicate event", func(t *testing.T) {
		defer cleanupOutboxHelper(ctx, store)()

		convID := models.ConversionID{
			Reference:   "conv-test-2",
			ConnectorID: defaultConnector.ID,
		}

		conv := models.Conversion{
			ID:           convID,
			ConnectorID:  defaultConnector.ID,
			Reference:    "conv-test-2",
			CreatedAt:    now.Add(-60 * time.Minute).UTC().Time,
			UpdatedAt:    now.Add(-5 * time.Minute).UTC().Time,
			SourceAsset:  "USD/2",
			DestinationAsset: "EUR/2",
			SourceAmount: big.NewInt(5000),
			Status:       models.CONVERSION_STATUS_PENDING,
			Metadata:     map[string]string{},
			Raw:          []byte(`{}`),
		}

		// Insert first time
		require.NoError(t, store.ConversionsUpsert(ctx, []models.Conversion{conv}))

		// Insert second time with same data
		conv.UpdatedAt = now.Add(-2 * time.Minute).UTC().Time
		require.NoError(t, store.ConversionsUpsert(ctx, []models.Conversion{conv}))

		// Verify only 1 outbox event (second was deduplicated via events_sent/outbox conflict)
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		convEvents := make([]models.OutboxEvent, 0)
		for _, event := range pendingEvents {
			if event.EventType == events.EventTypeSavedConversion && event.EntityID == convID.String() {
				convEvents = append(convEvents, event)
			}
		}
		require.Len(t, convEvents, 1, "expected 1 outbox event (duplicate was deduplicated)")
	})
}

func defaultConversions() []models.Conversion {
	convID1 := models.ConversionID{Reference: "conv-1", ConnectorID: defaultConnector.ID}
	convID2 := models.ConversionID{Reference: "conv-2", ConnectorID: defaultConnector.ID}

	return []models.Conversion{
		{
			ID:                convID1,
			ConnectorID:       defaultConnector.ID,
			Reference:         "conv-1",
			CreatedAt:         now.Add(-60 * time.Minute).UTC().Time,
			UpdatedAt:         now.Add(-50 * time.Minute).UTC().Time,
			SourceAsset:       "USD/2",
			DestinationAsset:  "USDC/6",
			SourceAmount:      big.NewInt(10000),
			DestinationAmount: big.NewInt(10000000),
			Status:            models.CONVERSION_STATUS_COMPLETED,
			Fee:               big.NewInt(50),
			FeeAsset:          pointer.For("USD/2"),
			Metadata:          map[string]string{"key1": "value1"},
			Raw:               []byte(`{}`),
		},
		{
			ID:               convID2,
			ConnectorID:      defaultConnector.ID,
			Reference:        "conv-2",
			CreatedAt:        now.Add(-30 * time.Minute).UTC().Time,
			UpdatedAt:        now.Add(-20 * time.Minute).UTC().Time,
			SourceAsset:      "EUR/2",
			DestinationAsset: "USD/2",
			SourceAmount:     big.NewInt(5000),
			Status:           models.CONVERSION_STATUS_PENDING,
			Metadata:         map[string]string{"key2": "value2"},
			Raw:              []byte(`{}`),
		},
	}
}

func upsertConversions(t *testing.T, ctx context.Context, store Storage, conversions []models.Conversion) {
	t.Helper()
	require.NoError(t, store.ConversionsUpsert(ctx, conversions))
}

func TestConversionsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	conversions := defaultConversions()
	upsertConversions(t, ctx, store, conversions)

	t.Run("get unknown conversion", func(t *testing.T) {
		unknownID := models.ConversionID{Reference: "unknown", ConnectorID: defaultConnector.ID}
		_, err := store.ConversionsGet(ctx, unknownID)
		require.Error(t, err)
	})

	t.Run("get existing conversion", func(t *testing.T) {
		for _, expected := range conversions {
			stored, err := store.ConversionsGet(ctx, expected.ID)
			require.NoError(t, err)

			assert.Equal(t, expected.ID, stored.ID)
			assert.Equal(t, expected.ConnectorID, stored.ConnectorID)
			assert.Equal(t, expected.Reference, stored.Reference)
			assert.Equal(t, expected.SourceAsset, stored.SourceAsset)
			assert.Equal(t, expected.DestinationAsset, stored.DestinationAsset)
			assert.Equal(t, expected.Status, stored.Status)
			assert.Equal(t, 0, expected.SourceAmount.Cmp(stored.SourceAmount))
		}
	})
}

func TestConversionsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	conversions := defaultConversions()
	upsertConversions(t, ctx, store, conversions)

	t.Run("list all conversions", func(t *testing.T) {
		q := NewListConversionsQuery(bunpaginate.NewPaginatedQueryOptions(ConversionQuery{}).WithPageSize(10))
		cursor, err := store.ConversionsList(ctx, q)
		require.NoError(t, err)
		assert.Len(t, cursor.Data, 2)
	})

	t.Run("filter by reference", func(t *testing.T) {
		q := NewListConversionsQuery(
			bunpaginate.NewPaginatedQueryOptions(ConversionQuery{}).
				WithPageSize(10).
				WithQueryBuilder(query.Match("reference", "conv-1")),
		)
		cursor, err := store.ConversionsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		assert.Equal(t, "conv-1", cursor.Data[0].Reference)
	})

	t.Run("filter by reference with invalid operator", func(t *testing.T) {
		q := NewListConversionsQuery(
			bunpaginate.NewPaginatedQueryOptions(ConversionQuery{}).
				WithPageSize(10).
				WithQueryBuilder(query.Lt("reference", "conv-1")),
		)
		_, err := store.ConversionsList(ctx, q)
		require.Error(t, err)
	})

	t.Run("filter by status", func(t *testing.T) {
		q := NewListConversionsQuery(
			bunpaginate.NewPaginatedQueryOptions(ConversionQuery{}).
				WithPageSize(10).
				WithQueryBuilder(query.Match("status", "COMPLETED")),
		)
		cursor, err := store.ConversionsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		assert.Equal(t, "conv-1", cursor.Data[0].Reference)
	})

	t.Run("filter by source_asset", func(t *testing.T) {
		q := NewListConversionsQuery(
			bunpaginate.NewPaginatedQueryOptions(ConversionQuery{}).
				WithPageSize(10).
				WithQueryBuilder(query.Match("source_asset", "EUR/2")),
		)
		cursor, err := store.ConversionsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		assert.Equal(t, "conv-2", cursor.Data[0].Reference)
	})

	t.Run("filter by metadata", func(t *testing.T) {
		q := NewListConversionsQuery(
			bunpaginate.NewPaginatedQueryOptions(ConversionQuery{}).
				WithPageSize(10).
				WithQueryBuilder(query.Match("metadata[key1]", "value1")),
		)
		cursor, err := store.ConversionsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		assert.Equal(t, "conv-1", cursor.Data[0].Reference)
	})

	t.Run("filter by metadata with invalid operator", func(t *testing.T) {
		q := NewListConversionsQuery(
			bunpaginate.NewPaginatedQueryOptions(ConversionQuery{}).
				WithPageSize(10).
				WithQueryBuilder(query.Lt("metadata[key1]", "value1")),
		)
		_, err := store.ConversionsList(ctx, q)
		require.Error(t, err)
	})

	t.Run("filter by unknown key", func(t *testing.T) {
		q := NewListConversionsQuery(
			bunpaginate.NewPaginatedQueryOptions(ConversionQuery{}).
				WithPageSize(10).
				WithQueryBuilder(query.Match("unknown_field", "value")),
		)
		_, err := store.ConversionsList(ctx, q)
		require.Error(t, err)
	})

	t.Run("pagination", func(t *testing.T) {
		q := NewListConversionsQuery(bunpaginate.NewPaginatedQueryOptions(ConversionQuery{}).WithPageSize(1))
		cursor, err := store.ConversionsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		assert.True(t, cursor.HasMore)

		// Next page
		var next ListConversionsQuery
		err = bunpaginate.UnmarshalCursor(cursor.Next, &next)
		require.NoError(t, err)
		cursor, err = store.ConversionsList(ctx, next)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		assert.False(t, cursor.HasMore)
	})
}

func TestConversionsDeleteFromConnectorID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	conversions := defaultConversions()
	upsertConversions(t, ctx, store, conversions)

	t.Run("delete from unknown connector", func(t *testing.T) {
		unknownConnID := models.ConnectorID{Reference: defaultConnector.ID.Reference, Provider: "unknown"}
		require.NoError(t, store.ConversionsDeleteFromConnectorID(ctx, unknownConnID))

		// Conversions should still exist
		for _, c := range conversions {
			_, err := store.ConversionsGet(ctx, c.ID)
			require.NoError(t, err)
		}
	})

	t.Run("delete from existing connector", func(t *testing.T) {
		require.NoError(t, store.ConversionsDeleteFromConnectorID(ctx, defaultConnector.ID))

		// Conversions should be gone
		for _, c := range conversions {
			_, err := store.ConversionsGet(ctx, c.ID)
			require.Error(t, err, fmt.Sprintf("conversion %s should have been deleted", c.ID.String()))
		}
	})
}
