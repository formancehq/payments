package storage

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
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
