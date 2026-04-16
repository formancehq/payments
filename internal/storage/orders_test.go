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

func defaultOrders() []models.Order {
	orderID1 := models.OrderID{Reference: "order-1", ConnectorID: defaultConnector.ID}
	orderID2 := models.OrderID{Reference: "order-2", ConnectorID: defaultConnector.ID}

	return []models.Order{
		{
			ID:                  orderID1,
			ConnectorID:         defaultConnector.ID,
			Reference:           "order-1",
			CreatedAt:           now.Add(-60 * time.Minute).UTC().Time,
			UpdatedAt:           now.Add(-50 * time.Minute).UTC().Time,
			Direction:           models.ORDER_DIRECTION_BUY,
			SourceAsset:         "USD/2",
			DestinationAsset:    "BTC/8",
			Type:                models.ORDER_TYPE_MARKET,
			Status:              models.ORDER_STATUS_FILLED,
			BaseQuantityOrdered: big.NewInt(100),
			BaseQuantityFilled:  big.NewInt(100),
			TimeInForce:         models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
			Fee:                 big.NewInt(10),
			FeeAsset:            pointer.For("USD/2"),
			Metadata:            map[string]string{"key1": "value1"},
			Adjustments: []models.OrderAdjustment{
				{
					ID: models.OrderAdjustmentID{
						OrderID:            orderID1,
						Reference:          "order-1",
						Status:             models.ORDER_STATUS_FILLED,
						BaseQuantityFilled: big.NewInt(100),
						Fee:                big.NewInt(10),
						FeeAsset:           pointer.For("USD/2"),
					},
					Reference:          "order-1",
					CreatedAt:          now.Add(-50 * time.Minute).UTC().Time,
					Status:             models.ORDER_STATUS_FILLED,
					BaseQuantityFilled: big.NewInt(100),
					Fee:                big.NewInt(10),
					FeeAsset:           pointer.For("USD/2"),
					Metadata:           map[string]string{},
					Raw:                []byte(`{}`),
				},
			},
		},
		{
			ID:                  orderID2,
			ConnectorID:         defaultConnector.ID,
			Reference:           "order-2",
			CreatedAt:           now.Add(-30 * time.Minute).UTC().Time,
			UpdatedAt:           now.Add(-20 * time.Minute).UTC().Time,
			Direction:           models.ORDER_DIRECTION_SELL,
			SourceAsset:         "BTC/8",
			DestinationAsset:    "EUR/2",
			Type:                models.ORDER_TYPE_LIMIT,
			Status:              models.ORDER_STATUS_OPEN,
			BaseQuantityOrdered: big.NewInt(200),
			BaseQuantityFilled:  big.NewInt(50),
			LimitPrice:          big.NewInt(5000000),
			TimeInForce:         models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
			Metadata:            map[string]string{"key2": "value2"},
			Adjustments: []models.OrderAdjustment{
				{
					ID: models.OrderAdjustmentID{
						OrderID:            orderID2,
						Reference:          "order-2",
						Status:             models.ORDER_STATUS_OPEN,
						BaseQuantityFilled: big.NewInt(50),
					},
					Reference:          "order-2",
					CreatedAt:          now.Add(-20 * time.Minute).UTC().Time,
					Status:             models.ORDER_STATUS_OPEN,
					BaseQuantityFilled: big.NewInt(50),
					Metadata:           map[string]string{},
					Raw:                []byte(`{}`),
				},
			},
		},
	}
}

func upsertOrders(t *testing.T, ctx context.Context, store Storage, orders []models.Order) {
	t.Helper()
	require.NoError(t, store.OrdersUpsert(ctx, orders))
}

func TestOrdersGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	orders := defaultOrders()
	upsertOrders(t, ctx, store, orders)

	t.Run("get unknown order", func(t *testing.T) {
		unknownID := models.OrderID{Reference: "unknown", ConnectorID: defaultConnector.ID}
		_, err := store.OrdersGet(ctx, unknownID)
		require.Error(t, err)
	})

	t.Run("get existing order", func(t *testing.T) {
		for _, expected := range orders {
			stored, err := store.OrdersGet(ctx, expected.ID)
			require.NoError(t, err)

			assert.Equal(t, expected.ID, stored.ID)
			assert.Equal(t, expected.ConnectorID, stored.ConnectorID)
			assert.Equal(t, expected.Reference, stored.Reference)
			assert.Equal(t, expected.Direction, stored.Direction)
			assert.Equal(t, expected.SourceAsset, stored.SourceAsset)
			assert.Equal(t, expected.DestinationAsset, stored.DestinationAsset)
			assert.Equal(t, expected.Type, stored.Type)
			assert.Equal(t, expected.TimeInForce, stored.TimeInForce)
			assert.Equal(t, 0, expected.BaseQuantityOrdered.Cmp(stored.BaseQuantityOrdered))
			require.Len(t, stored.Adjustments, len(expected.Adjustments))
		}
	})
}

func TestOrdersListSorting(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)

	orderID := models.OrderID{Reference: "order-sort", ConnectorID: defaultConnector.ID}

	// Create order with 2 adjustments: PENDING then FILLED
	order := models.Order{
		ID:                  orderID,
		ConnectorID:         defaultConnector.ID,
		Reference:           "order-sort",
		CreatedAt:           now.Add(-60 * time.Minute).UTC().Time,
		UpdatedAt:           now.Add(-10 * time.Minute).UTC().Time,
		Direction:           models.ORDER_DIRECTION_BUY,
		SourceAsset:         "USD/2",
		DestinationAsset:    "BTC/8",
		Type:                models.ORDER_TYPE_MARKET,
		Status:              models.ORDER_STATUS_FILLED,
		BaseQuantityOrdered: big.NewInt(100),
		BaseQuantityFilled:  big.NewInt(100),
		TimeInForce:         models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
		Metadata:            map[string]string{},
		Adjustments: []models.OrderAdjustment{
			{
				ID: models.OrderAdjustmentID{
					OrderID:   orderID,
					Reference: "order-sort",
					Status:    models.ORDER_STATUS_PENDING,
				},
				Reference: "order-sort",
				CreatedAt: now.Add(-30 * time.Minute).UTC().Time,
				Status:    models.ORDER_STATUS_PENDING,
				Metadata:  map[string]string{},
				Raw:       []byte(`{}`),
			},
			{
				ID: models.OrderAdjustmentID{
					OrderID:            orderID,
					Reference:          "order-sort",
					Status:             models.ORDER_STATUS_FILLED,
					BaseQuantityFilled: big.NewInt(100),
				},
				Reference:          "order-sort",
				CreatedAt:          now.Add(-10 * time.Minute).UTC().Time,
				Status:             models.ORDER_STATUS_FILLED,
				BaseQuantityFilled: big.NewInt(100),
				Metadata:           map[string]string{},
				Raw:                []byte(`{}`),
			},
		},
	}

	upsertOrders(t, ctx, store, []models.Order{order})

	q := NewListOrdersQuery(bunpaginate.NewPaginatedQueryOptions(OrderQuery{}).WithPageSize(1))
	cursor, err := store.OrdersList(ctx, q)
	require.NoError(t, err)
	require.Len(t, cursor.Data, 1)

	// Status should be FILLED (latest adjustment via lateral join)
	assert.Equal(t, models.ORDER_STATUS_FILLED, cursor.Data[0].Status)
}

func TestOrdersList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	orders := defaultOrders()
	upsertOrders(t, ctx, store, orders)

	t.Run("list all orders", func(t *testing.T) {
		q := NewListOrdersQuery(bunpaginate.NewPaginatedQueryOptions(OrderQuery{}).WithPageSize(10))
		cursor, err := store.OrdersList(ctx, q)
		require.NoError(t, err)
		assert.Len(t, cursor.Data, 2)
	})

	t.Run("filter by reference", func(t *testing.T) {
		q := NewListOrdersQuery(
			bunpaginate.NewPaginatedQueryOptions(OrderQuery{}).
				WithPageSize(10).
				WithQueryBuilder(query.Match("reference", "order-1")),
		)
		cursor, err := store.OrdersList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		assert.Equal(t, "order-1", cursor.Data[0].Reference)
	})

	t.Run("filter by reference with invalid operator", func(t *testing.T) {
		q := NewListOrdersQuery(
			bunpaginate.NewPaginatedQueryOptions(OrderQuery{}).
				WithPageSize(10).
				WithQueryBuilder(query.Lt("reference", "order-1")),
		)
		_, err := store.OrdersList(ctx, q)
		require.Error(t, err)
	})

	t.Run("filter by direction", func(t *testing.T) {
		q := NewListOrdersQuery(
			bunpaginate.NewPaginatedQueryOptions(OrderQuery{}).
				WithPageSize(10).
				WithQueryBuilder(query.Match("direction", "BUY")),
		)
		cursor, err := store.OrdersList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		assert.Equal(t, models.ORDER_DIRECTION_BUY, cursor.Data[0].Direction)
	})

	t.Run("filter by status", func(t *testing.T) {
		q := NewListOrdersQuery(
			bunpaginate.NewPaginatedQueryOptions(OrderQuery{}).
				WithPageSize(10).
				WithQueryBuilder(query.Match("status", "FILLED")),
		)
		cursor, err := store.OrdersList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		assert.Equal(t, "order-1", cursor.Data[0].Reference)
	})

	t.Run("filter by type", func(t *testing.T) {
		q := NewListOrdersQuery(
			bunpaginate.NewPaginatedQueryOptions(OrderQuery{}).
				WithPageSize(10).
				WithQueryBuilder(query.Match("type", "LIMIT")),
		)
		cursor, err := store.OrdersList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		assert.Equal(t, "order-2", cursor.Data[0].Reference)
	})

	t.Run("filter by metadata", func(t *testing.T) {
		q := NewListOrdersQuery(
			bunpaginate.NewPaginatedQueryOptions(OrderQuery{}).
				WithPageSize(10).
				WithQueryBuilder(query.Match("metadata[key1]", "value1")),
		)
		cursor, err := store.OrdersList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		assert.Equal(t, "order-1", cursor.Data[0].Reference)
	})

	t.Run("filter by metadata with invalid operator", func(t *testing.T) {
		q := NewListOrdersQuery(
			bunpaginate.NewPaginatedQueryOptions(OrderQuery{}).
				WithPageSize(10).
				WithQueryBuilder(query.Lt("metadata[key1]", "value1")),
		)
		_, err := store.OrdersList(ctx, q)
		require.Error(t, err)
	})

	t.Run("filter by unknown key", func(t *testing.T) {
		q := NewListOrdersQuery(
			bunpaginate.NewPaginatedQueryOptions(OrderQuery{}).
				WithPageSize(10).
				WithQueryBuilder(query.Match("unknown_field", "value")),
		)
		_, err := store.OrdersList(ctx, q)
		require.Error(t, err)
	})

	t.Run("pagination", func(t *testing.T) {
		q := NewListOrdersQuery(bunpaginate.NewPaginatedQueryOptions(OrderQuery{}).WithPageSize(1))
		cursor, err := store.OrdersList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		assert.True(t, cursor.HasMore)

		// Next page
		var next ListOrdersQuery
		err = bunpaginate.UnmarshalCursor(cursor.Next, &next)
		require.NoError(t, err)
		cursor, err = store.OrdersList(ctx, next)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		assert.False(t, cursor.HasMore)
	})
}

func TestOrdersDeleteFromConnectorID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	orders := defaultOrders()
	upsertOrders(t, ctx, store, orders)

	t.Run("delete from unknown connector", func(t *testing.T) {
		unknownConnID := models.ConnectorID{Reference: defaultConnector.ID.Reference, Provider: "unknown"}
		require.NoError(t, store.OrdersDeleteFromConnectorID(ctx, unknownConnID))

		// Orders should still exist
		for _, o := range orders {
			_, err := store.OrdersGet(ctx, o.ID)
			require.NoError(t, err)
		}
	})

	t.Run("delete from existing connector", func(t *testing.T) {
		require.NoError(t, store.OrdersDeleteFromConnectorID(ctx, defaultConnector.ID))

		// Orders should be gone
		for _, o := range orders {
			_, err := store.OrdersGet(ctx, o.ID)
			require.Error(t, err, fmt.Sprintf("order %s should have been deleted", o.ID.String()))
		}
	})
}
