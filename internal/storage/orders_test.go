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

func TestOrdersUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)

	orderID := models.OrderID{
		Reference:   "order-test-1",
		ConnectorID: defaultConnector.ID,
	}

	t.Run("different fill quantities produce distinct adjustments and events", func(t *testing.T) {
		defer cleanupOutboxHelper(ctx, store)()

		observedAt1 := now.Add(-10 * time.Minute).UTC().Time
		observedAt2 := now.Add(-5 * time.Minute).UTC().Time

		// First fetch: order at 50% fill
		order1 := models.Order{
			ID:                  orderID,
			ConnectorID:         defaultConnector.ID,
			Reference:           "order-test-1",
			CreatedAt:           now.Add(-60 * time.Minute).UTC().Time,
			UpdatedAt:           observedAt1,
			Direction:           models.ORDER_DIRECTION_BUY,
			SourceAsset:         "USD/2",
			DestinationAsset:    "BTC/8",
			Type:                models.ORDER_TYPE_MARKET,
			Status:              models.ORDER_STATUS_PARTIALLY_FILLED,
			BaseQuantityOrdered: big.NewInt(100),
			BaseQuantityFilled:  big.NewInt(50),
			TimeInForce:         models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
			Fee:                 big.NewInt(10),
			FeeAsset:            pointer.For("USD/2"),
			Metadata:            map[string]string{},
			Adjustments: []models.OrderAdjustment{
				{
					ID: models.OrderAdjustmentID{
						OrderID:            orderID,
						Reference:          "order-test-1",
						Status:             models.ORDER_STATUS_PARTIALLY_FILLED,
						BaseQuantityFilled: big.NewInt(50),
						Fee:                big.NewInt(10),
						FeeAsset:           pointer.For("USD/2"),
					},
					Reference:          "order-test-1",
					CreatedAt:          observedAt1,
					Status:             models.ORDER_STATUS_PARTIALLY_FILLED,
					BaseQuantityFilled: big.NewInt(50),
					Fee:                big.NewInt(10),
					FeeAsset:           pointer.For("USD/2"),
					Metadata:           map[string]string{},
					Raw:                []byte(`{"fill": "50%"}`),
				},
			},
		}

		require.NoError(t, store.OrdersUpsert(ctx, []models.Order{order1}))

		// Second fetch: same order at 75% fill
		order2 := models.Order{
			ID:                  orderID,
			ConnectorID:         defaultConnector.ID,
			Reference:           "order-test-1",
			CreatedAt:           now.Add(-60 * time.Minute).UTC().Time,
			UpdatedAt:           observedAt2,
			Direction:           models.ORDER_DIRECTION_BUY,
			SourceAsset:         "USD/2",
			DestinationAsset:    "BTC/8",
			Type:                models.ORDER_TYPE_MARKET,
			Status:              models.ORDER_STATUS_PARTIALLY_FILLED,
			BaseQuantityOrdered: big.NewInt(100),
			BaseQuantityFilled:  big.NewInt(75),
			TimeInForce:         models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
			Fee:                 big.NewInt(15),
			FeeAsset:            pointer.For("USD/2"),
			Metadata:            map[string]string{},
			Adjustments: []models.OrderAdjustment{
				{
					ID: models.OrderAdjustmentID{
						OrderID:            orderID,
						Reference:          "order-test-1",
						Status:             models.ORDER_STATUS_PARTIALLY_FILLED,
						BaseQuantityFilled: big.NewInt(75),
						Fee:                big.NewInt(15),
						FeeAsset:           pointer.For("USD/2"),
					},
					Reference:          "order-test-1",
					CreatedAt:          observedAt2,
					Status:             models.ORDER_STATUS_PARTIALLY_FILLED,
					BaseQuantityFilled: big.NewInt(75),
					Fee:                big.NewInt(15),
					FeeAsset:           pointer.For("USD/2"),
					Metadata:           map[string]string{},
					Raw:                []byte(`{"fill": "75%"}`),
				},
			},
		}

		require.NoError(t, store.OrdersUpsert(ctx, []models.Order{order2}))

		// Verify: 2 distinct adjustments stored
		storedOrder, err := store.OrdersGet(ctx, orderID)
		require.NoError(t, err)
		require.Len(t, storedOrder.Adjustments, 2, "expected 2 distinct adjustments for different fill quantities")

		// Verify: 2 distinct outbox events
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		orderEvents := make([]models.OutboxEvent, 0)
		for _, event := range pendingEvents {
			if event.EventType == events.EventTypeSavedOrder && event.EntityID == orderID.String() {
				orderEvents = append(orderEvents, event)
			}
		}
		require.Len(t, orderEvents, 2, "expected 2 outbox events for 2 distinct adjustments")
	})

	t.Run("identical data deduplicates adjustment and event", func(t *testing.T) {
		defer cleanupOutboxHelper(ctx, store)()

		orderID2 := models.OrderID{
			Reference:   "order-test-2",
			ConnectorID: defaultConnector.ID,
		}
		observedAt := now.Add(-10 * time.Minute).UTC().Time

		order := models.Order{
			ID:                  orderID2,
			ConnectorID:         defaultConnector.ID,
			Reference:           "order-test-2",
			CreatedAt:           now.Add(-60 * time.Minute).UTC().Time,
			UpdatedAt:           observedAt,
			Direction:           models.ORDER_DIRECTION_SELL,
			SourceAsset:         "BTC/8",
			DestinationAsset:    "USD/2",
			Type:                models.ORDER_TYPE_LIMIT,
			Status:              models.ORDER_STATUS_PARTIALLY_FILLED,
			BaseQuantityOrdered: big.NewInt(200),
			BaseQuantityFilled:  big.NewInt(100),
			TimeInForce:         models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
			Fee:                 big.NewInt(5),
			FeeAsset:            pointer.For("USD/2"),
			Metadata:            map[string]string{},
			Adjustments: []models.OrderAdjustment{
				{
					ID: models.OrderAdjustmentID{
						OrderID:            orderID2,
						Reference:          "order-test-2",
						Status:             models.ORDER_STATUS_PARTIALLY_FILLED,
						BaseQuantityFilled: big.NewInt(100),
						Fee:                big.NewInt(5),
						FeeAsset:           pointer.For("USD/2"),
					},
					Reference:          "order-test-2",
					CreatedAt:          observedAt,
					Status:             models.ORDER_STATUS_PARTIALLY_FILLED,
					BaseQuantityFilled: big.NewInt(100),
					Fee:                big.NewInt(5),
					FeeAsset:           pointer.For("USD/2"),
					Metadata:           map[string]string{},
					Raw:                []byte(`{}`),
				},
			},
		}

		// Insert first time
		require.NoError(t, store.OrdersUpsert(ctx, []models.Order{order}))

		// Insert second time with identical data
		order.UpdatedAt = now.Add(-4 * time.Minute).UTC().Time // UpdatedAt changes but adjustment is identical
		require.NoError(t, store.OrdersUpsert(ctx, []models.Order{order}))

		// Verify: only 1 adjustment
		storedOrder, err := store.OrdersGet(ctx, orderID2)
		require.NoError(t, err)
		require.Len(t, storedOrder.Adjustments, 1, "expected 1 adjustment (second was deduplicated)")

		// Verify: only 1 outbox event
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		orderEvents := make([]models.OutboxEvent, 0)
		for _, event := range pendingEvents {
			if event.EventType == events.EventTypeSavedOrder && event.EntityID == orderID2.String() {
				orderEvents = append(orderEvents, event)
			}
		}
		require.Len(t, orderEvents, 1, "expected 1 outbox event (duplicate was deduplicated)")
	})

	t.Run("adjustment and outbox event are atomic", func(t *testing.T) {
		defer cleanupOutboxHelper(ctx, store)()

		orderID3 := models.OrderID{
			Reference:   "order-test-3",
			ConnectorID: defaultConnector.ID,
		}
		observedAt := now.Add(-10 * time.Minute).UTC().Time

		order := models.Order{
			ID:                  orderID3,
			ConnectorID:         defaultConnector.ID,
			Reference:           "order-test-3",
			CreatedAt:           now.Add(-60 * time.Minute).UTC().Time,
			UpdatedAt:           observedAt,
			Direction:           models.ORDER_DIRECTION_BUY,
			SourceAsset:         "USD/2",
			DestinationAsset:    "ETH/8",
			Type:                models.ORDER_TYPE_MARKET,
			Status:              models.ORDER_STATUS_FILLED,
			BaseQuantityOrdered: big.NewInt(50),
			BaseQuantityFilled:  big.NewInt(50),
			TimeInForce:         models.TIME_IN_FORCE_FILL_OR_KILL,
			Fee:                 big.NewInt(2),
			FeeAsset:            pointer.For("USD/2"),
			Metadata:            map[string]string{},
			Adjustments: []models.OrderAdjustment{
				{
					ID: models.OrderAdjustmentID{
						OrderID:            orderID3,
						Reference:          "order-test-3",
						Status:             models.ORDER_STATUS_FILLED,
						BaseQuantityFilled: big.NewInt(50),
						Fee:                big.NewInt(2),
						FeeAsset:           pointer.For("USD/2"),
					},
					Reference:          "order-test-3",
					CreatedAt:          observedAt,
					Status:             models.ORDER_STATUS_FILLED,
					BaseQuantityFilled: big.NewInt(50),
					Fee:                big.NewInt(2),
					FeeAsset:           pointer.For("USD/2"),
					Metadata:           map[string]string{},
					Raw:                []byte(`{"status": "filled"}`),
				},
			},
		}

		require.NoError(t, store.OrdersUpsert(ctx, []models.Order{order}))

		// Verify adjustment exists
		storedOrder, err := store.OrdersGet(ctx, orderID3)
		require.NoError(t, err)
		require.Len(t, storedOrder.Adjustments, 1)

		// Verify outbox event exists for the same adjustment
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		expectedKey := order.Adjustments[0].IdempotencyKey()
		var found bool
		for _, event := range pendingEvents {
			if event.ID.EventIdempotencyKey == expectedKey {
				found = true
				assert.Equal(t, events.EventTypeSavedOrder, event.EventType)
				assert.Equal(t, orderID3.String(), event.EntityID)
				assert.Equal(t, models.OUTBOX_STATUS_PENDING, event.Status)
				assert.Equal(t, defaultConnector.ID, *event.ConnectorID)

				// Verify payload
				var payload map[string]interface{}
				err = json.Unmarshal(event.Payload, &payload)
				require.NoError(t, err)
				assert.Equal(t, order.ID.String(), payload["id"])
				assert.Equal(t, order.Status.String(), payload["status"])
				break
			}
		}
		assert.True(t, found, "outbox event must exist for new adjustment")
	})

	t.Run("adjustment CreatedAt reflects observation time not order creation time", func(t *testing.T) {
		defer cleanupOutboxHelper(ctx, store)()

		orderID4 := models.OrderID{
			Reference:   "order-test-4",
			ConnectorID: defaultConnector.ID,
		}
		orderCreatedAt := now.Add(-60 * time.Minute).UTC().Time
		observedAt := now.Add(-2 * time.Minute).UTC().Time

		order := models.Order{
			ID:                  orderID4,
			ConnectorID:         defaultConnector.ID,
			Reference:           "order-test-4",
			CreatedAt:           orderCreatedAt,
			UpdatedAt:           observedAt,
			Direction:           models.ORDER_DIRECTION_BUY,
			SourceAsset:         "EUR/2",
			DestinationAsset:    "BTC/8",
			Type:                models.ORDER_TYPE_MARKET,
			Status:              models.ORDER_STATUS_OPEN,
			BaseQuantityOrdered: big.NewInt(100),
			TimeInForce:         models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
			Metadata:            map[string]string{},
			Adjustments: []models.OrderAdjustment{
				{
					ID: models.OrderAdjustmentID{
						OrderID:   orderID4,
						Reference: "order-test-4",
						Status:    models.ORDER_STATUS_OPEN,
					},
					Reference: "order-test-4",
					CreatedAt: observedAt, // observation time, not orderCreatedAt
					Status:    models.ORDER_STATUS_OPEN,
					Metadata:  map[string]string{},
					Raw:       []byte(`{}`),
				},
			},
		}

		require.NoError(t, store.OrdersUpsert(ctx, []models.Order{order}))

		storedOrder, err := store.OrdersGet(ctx, orderID4)
		require.NoError(t, err)
		require.Len(t, storedOrder.Adjustments, 1)

		// Adjustment CreatedAt should be the observation time, not the order creation time
		adj := storedOrder.Adjustments[0]
		assert.Equal(t, observedAt.Truncate(time.Microsecond), adj.CreatedAt.Truncate(time.Microsecond),
			"adjustment CreatedAt should be observation time")
		assert.NotEqual(t, orderCreatedAt.Truncate(time.Microsecond), adj.CreatedAt.Truncate(time.Microsecond),
			"adjustment CreatedAt should NOT be order creation time")
	})
}
