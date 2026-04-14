package events

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderMessagePayload_MarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("with all big.Int fields", func(t *testing.T) {
		t.Parallel()

		payload := OrderMessagePayload{
			ID:                  "order-1",
			ConnectorID:         "conn-1",
			Provider:            "coinbaseprime",
			Reference:           "ref-1",
			CreatedAt:           time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC),
			UpdatedAt:           time.Date(2026, 2, 24, 11, 0, 0, 0, time.UTC),
			Direction:           "BUY",
			SourceAsset:         "USD/2",
			DestinationAsset:    "BTC/8",
			Type:                "MARKET",
			Status:              "FILLED",
			TimeInForce:         "IMMEDIATE_OR_CANCEL",
			BaseQuantityOrdered: big.NewInt(50000000),
			BaseQuantityFilled:  big.NewInt(50000000),
			LimitPrice:          big.NewInt(5000000),
			StopPrice:           big.NewInt(4900000),
			QuoteAmount:         big.NewInt(10000),
			Fee:                 big.NewInt(14),
			AverageFillPrice:    big.NewInt(5000000),
		}

		data, err := json.Marshal(&payload)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "order-1", result["id"])
		assert.Equal(t, "50000000", result["baseQuantityOrdered"])
		assert.Equal(t, "50000000", result["baseQuantityFilled"])
		assert.Equal(t, "5000000", result["limitPrice"])
		assert.Equal(t, "4900000", result["stopPrice"])
		assert.Equal(t, "10000", result["quoteAmount"])
		assert.Equal(t, "14", result["fee"])
		assert.Equal(t, "5000000", result["averageFillPrice"])
	})

	t.Run("with nil big.Int fields", func(t *testing.T) {
		t.Parallel()

		payload := OrderMessagePayload{
			ID:                  "order-2",
			BaseQuantityOrdered: big.NewInt(100),
		}

		data, err := json.Marshal(&payload)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "100", result["baseQuantityOrdered"])
		assert.Nil(t, result["baseQuantityFilled"])
		assert.Nil(t, result["limitPrice"])
		assert.Nil(t, result["stopPrice"])
		assert.Nil(t, result["quoteAmount"])
		assert.Nil(t, result["fee"])
		assert.Nil(t, result["averageFillPrice"])
	})

	t.Run("round-trip", func(t *testing.T) {
		t.Parallel()

		feeAsset := "USD/2"
		priceAsset := "USD/6"
		original := OrderMessagePayload{
			ID:                  "order-rt",
			ConnectorID:         "conn-1",
			Provider:            "coinbaseprime",
			Reference:           "ref-rt",
			Direction:           "SELL",
			SourceAsset:         "BTC/8",
			DestinationAsset:    "USD/2",
			Type:                "LIMIT",
			Status:              "OPEN",
			TimeInForce:         "GOOD_UNTIL_CANCELLED",
			BaseQuantityOrdered: big.NewInt(150000000),
			BaseQuantityFilled:  big.NewInt(50000000),
			LimitPrice:          big.NewInt(6700000000),
			QuoteAmount:         big.NewInt(3371110),
			Fee:                 big.NewInt(1685),
			FeeAsset:            &feeAsset,
			AverageFillPrice:    big.NewInt(6742222000),
			PriceAsset:          &priceAsset,
			QuoteAsset:          "USD/2",
		}

		data, err := json.Marshal(&original)
		require.NoError(t, err)

		var restored OrderMessagePayload
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		assert.Equal(t, original.ID, restored.ID)
		assert.Equal(t, original.Direction, restored.Direction)
		assert.Equal(t, 0, original.BaseQuantityOrdered.Cmp(restored.BaseQuantityOrdered))
		assert.Equal(t, 0, original.BaseQuantityFilled.Cmp(restored.BaseQuantityFilled))
		assert.Equal(t, 0, original.LimitPrice.Cmp(restored.LimitPrice))
		assert.Equal(t, 0, original.QuoteAmount.Cmp(restored.QuoteAmount))
		assert.Equal(t, 0, original.Fee.Cmp(restored.Fee))
		assert.Equal(t, 0, original.AverageFillPrice.Cmp(restored.AverageFillPrice))
		assert.Equal(t, *original.FeeAsset, *restored.FeeAsset)
		assert.Equal(t, *original.PriceAsset, *restored.PriceAsset)
	})

	t.Run("with adjustments", func(t *testing.T) {
		t.Parallel()

		payload := OrderMessagePayload{
			ID:                  "order-adj",
			BaseQuantityOrdered: big.NewInt(100),
			Adjustments: []OrderAdjustmentPayload{
				{
					ID:        "adj-1",
					Reference: "ref-1",
					Status:    "OPEN",
					Raw:       json.RawMessage(`{"key":"value"}`),
				},
			},
		}

		data, err := json.Marshal(&payload)
		require.NoError(t, err)

		var restored OrderMessagePayload
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		require.Len(t, restored.Adjustments, 1)
		assert.Equal(t, "adj-1", restored.Adjustments[0].ID)
		assert.Equal(t, "OPEN", restored.Adjustments[0].Status)
	})
}

func TestNewEventSavedOrder(t *testing.T) {
	t.Parallel()

	connID := models.ConnectorID{Provider: "test", Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001")}
	order := models.Order{
		ID:                  models.OrderID{Reference: "ord-1", ConnectorID: connID},
		ConnectorID:         connID,
		Reference:           "ord-1",
		CreatedAt:           time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:           time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Direction:           models.ORDER_DIRECTION_BUY,
		SourceAsset:         "USD/2",
		DestinationAsset:    "BTC/8",
		Type:                models.ORDER_TYPE_MARKET,
		Status:              models.ORDER_STATUS_FILLED,
		TimeInForce:         models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL,
		BaseQuantityOrdered: big.NewInt(50000000),
		BaseQuantityFilled:  big.NewInt(50000000),
		QuoteAmount:         big.NewInt(10000),
		QuoteAsset:          "USD/2",
		Fee:                 big.NewInt(14),
		Metadata:            map[string]string{"key": "val"},
		Adjustments: []models.OrderAdjustment{
			{
				ID:        models.OrderAdjustmentID{Reference: "adj-1", Status: models.ORDER_STATUS_FILLED},
				Reference: "adj-1",
				CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				Status:    models.ORDER_STATUS_FILLED,
				Raw:       json.RawMessage(`{"raw":"data"}`),
			},
		},
	}

	e := Events{}
	msg := e.NewEventSavedOrder(order, order.Adjustments[0])

	assert.Equal(t, "SAVED_ORDER", msg.Type)
	assert.NotEmpty(t, msg.IdempotencyKey)

	payload, ok := msg.Payload.(OrderMessagePayload)
	require.True(t, ok)
	assert.Equal(t, "ord-1", payload.Reference)
	assert.Equal(t, "BUY", payload.Direction)
	assert.Equal(t, "USD/2", payload.SourceAsset)
	assert.Equal(t, "BTC/8", payload.DestinationAsset)
	assert.Equal(t, 0, payload.BaseQuantityOrdered.Cmp(big.NewInt(50000000)))
	assert.Equal(t, 0, payload.QuoteAmount.Cmp(big.NewInt(10000)))
	assert.Len(t, payload.Adjustments, 1)
	assert.Equal(t, "adj-1", payload.Adjustments[0].Reference)
}
