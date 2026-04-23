package models_test

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newConnectorID(t *testing.T) models.ConnectorID {
	t.Helper()
	return models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "coinbaseprime",
	}
}

func validPSPOrder() models.PSPOrder {
	return models.PSPOrder{
		Reference:           "order-ref-1",
		ClientOrderID:       "client-order-1",
		CreatedAt:           time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Direction:           models.ORDER_DIRECTION_BUY,
		SourceAsset:         "USD/2",
		DestinationAsset:    "BTC/8",
		Type:                models.ORDER_TYPE_LIMIT,
		Status:              models.ORDER_STATUS_FILLED,
		BaseQuantityOrdered: big.NewInt(1000),
		BaseQuantityFilled:  big.NewInt(1000),
		LimitPrice:          big.NewInt(5000000),
		TimeInForce:         models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
		QuoteAmount:         big.NewInt(5000000000),
		QuoteAsset:          "USD/2",
		Fee:                 big.NewInt(100),
		FeeAsset:            pointer.For("USD/2"),
		AverageFillPrice:    big.NewInt(5000000),
		PriceAsset:          pointer.For("USD/2"),
		SourceAccountReference:      pointer.For("src-wallet"),
		DestinationAccountReference: pointer.For("dst-wallet"),
		Metadata: map[string]string{
			"k": "v",
		},
		Raw: json.RawMessage(`{"raw":"ok"}`),
	}
}

func TestPSPOrderValidate(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		o := validPSPOrder()
		assert.NoError(t, o.Validate())
	})

	cases := []struct {
		name   string
		mutate func(*models.PSPOrder)
		errMsg string
	}{
		{"missing reference", func(o *models.PSPOrder) { o.Reference = "" }, "missing order reference"},
		{"missing createdAt", func(o *models.PSPOrder) { o.CreatedAt = time.Time{} }, "missing order createdAt"},
		{"missing direction", func(o *models.PSPOrder) { o.Direction = models.ORDER_DIRECTION_UNKNOWN }, "missing order direction"},
		{"invalid source asset", func(o *models.PSPOrder) { o.SourceAsset = "nope" }, "invalid order source asset"},
		{"invalid dest asset", func(o *models.PSPOrder) { o.DestinationAsset = "nope" }, "invalid order target asset"},
		{"missing type", func(o *models.PSPOrder) { o.Type = models.ORDER_TYPE_UNKNOWN }, "missing order type"},
		{"missing status", func(o *models.PSPOrder) { o.Status = models.ORDER_STATUS_UNKNOWN }, "missing order status"},
		{"missing base quantity ordered", func(o *models.PSPOrder) { o.BaseQuantityOrdered = nil }, "missing order base quantity ordered"},
		{"missing raw", func(o *models.PSPOrder) { o.Raw = nil }, "missing order raw"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			o := validPSPOrder()
			tc.mutate(&o)
			err := o.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

func TestFromPSPOrderToOrder(t *testing.T) {
	t.Parallel()
	connectorID := newConnectorID(t)
	observedAt := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		psp := validPSPOrder()

		order, err := models.FromPSPOrderToOrder(psp, connectorID, observedAt)
		require.NoError(t, err)

		assert.Equal(t, psp.Reference, order.Reference)
		assert.Equal(t, psp.ClientOrderID, order.ClientOrderID)
		assert.Equal(t, psp.CreatedAt, order.CreatedAt)
		assert.Equal(t, observedAt, order.UpdatedAt)
		assert.Equal(t, connectorID, order.ConnectorID)
		assert.Equal(t, psp.SourceAsset, order.SourceAsset)
		assert.Equal(t, psp.DestinationAsset, order.DestinationAsset)
		assert.Equal(t, psp.Status, order.Status)
		assert.Equal(t, psp.BaseQuantityOrdered, order.BaseQuantityOrdered)
		require.NotNil(t, order.SourceAccountID)
		require.NotNil(t, order.DestinationAccountID)
		assert.Equal(t, "src-wallet", order.SourceAccountID.Reference)
		assert.Equal(t, "dst-wallet", order.DestinationAccountID.Reference)

		require.Len(t, order.Adjustments, 1)
		adj := order.Adjustments[0]
		assert.Equal(t, psp.Reference, adj.Reference)
		assert.Equal(t, psp.Status, adj.Status)
		assert.Equal(t, observedAt, adj.CreatedAt)
		assert.Equal(t, json.RawMessage(`{"raw":"ok"}`), adj.Raw)
	})

	t.Run("nil account references produce nil account ids", func(t *testing.T) {
		t.Parallel()
		psp := validPSPOrder()
		psp.SourceAccountReference = nil
		psp.DestinationAccountReference = nil

		order, err := models.FromPSPOrderToOrder(psp, connectorID, observedAt)
		require.NoError(t, err)
		assert.Nil(t, order.SourceAccountID)
		assert.Nil(t, order.DestinationAccountID)
	})

	t.Run("invalid order returns error", func(t *testing.T) {
		t.Parallel()
		psp := validPSPOrder()
		psp.Reference = ""

		_, err := models.FromPSPOrderToOrder(psp, connectorID, observedAt)
		require.Error(t, err)
	})
}

func TestFromPSPOrders(t *testing.T) {
	t.Parallel()
	connectorID := newConnectorID(t)
	observedAt := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	t.Run("all valid", func(t *testing.T) {
		t.Parallel()
		p1 := validPSPOrder()
		p1.Reference = "a"
		p2 := validPSPOrder()
		p2.Reference = "b"

		orders, err := models.FromPSPOrders([]models.PSPOrder{p1, p2}, connectorID, observedAt)
		require.NoError(t, err)
		require.Len(t, orders, 2)
		assert.Equal(t, "a", orders[0].Reference)
		assert.Equal(t, "b", orders[1].Reference)
	})

	t.Run("empty slice", func(t *testing.T) {
		t.Parallel()
		orders, err := models.FromPSPOrders(nil, connectorID, observedAt)
		require.NoError(t, err)
		assert.Empty(t, orders)
	})

	t.Run("one invalid aborts", func(t *testing.T) {
		t.Parallel()
		good := validPSPOrder()
		bad := validPSPOrder()
		bad.Reference = ""

		_, err := models.FromPSPOrders([]models.PSPOrder{good, bad}, connectorID, observedAt)
		require.Error(t, err)
	})
}

func TestToPSPOrder(t *testing.T) {
	t.Parallel()
	connectorID := newConnectorID(t)
	observedAt := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	t.Run("round trip preserves fields", func(t *testing.T) {
		t.Parallel()
		original := validPSPOrder()
		order, err := models.FromPSPOrderToOrder(original, connectorID, observedAt)
		require.NoError(t, err)

		psp := models.ToPSPOrder(&order)
		assert.Equal(t, original.Reference, psp.Reference)
		assert.Equal(t, original.SourceAsset, psp.SourceAsset)
		assert.Equal(t, original.DestinationAsset, psp.DestinationAsset)
		assert.Equal(t, original.Status, psp.Status)
		assert.Equal(t, original.BaseQuantityOrdered, psp.BaseQuantityOrdered)
		require.NotNil(t, psp.SourceAccountReference)
		assert.Equal(t, "src-wallet", *psp.SourceAccountReference)
		require.NotNil(t, psp.DestinationAccountReference)
		assert.Equal(t, "dst-wallet", *psp.DestinationAccountReference)

		assert.NoError(t, psp.Validate())
	})

	t.Run("picks latest adjustment Raw", func(t *testing.T) {
		t.Parallel()
		oldAdj := models.OrderAdjustment{
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Raw:       json.RawMessage(`{"v":"old"}`),
		}
		newAdj := models.OrderAdjustment{
			CreatedAt: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			Raw:       json.RawMessage(`{"v":"new"}`),
		}
		order := models.Order{
			Reference:   "x",
			ConnectorID: connectorID,
			Adjustments: []models.OrderAdjustment{oldAdj, newAdj},
		}
		psp := models.ToPSPOrder(&order)
		assert.Equal(t, json.RawMessage(`{"v":"new"}`), psp.Raw)
	})

	t.Run("empty adjustments -> nil raw", func(t *testing.T) {
		t.Parallel()
		order := models.Order{
			Reference:   "x",
			ConnectorID: connectorID,
		}
		psp := models.ToPSPOrder(&order)
		assert.Nil(t, psp.Raw)
	})
}

func TestOrderMarshalUnmarshal(t *testing.T) {
	t.Parallel()
	connectorID := newConnectorID(t)
	observedAt := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	order, err := models.FromPSPOrderToOrder(validPSPOrder(), connectorID, observedAt)
	require.NoError(t, err)

	data, err := json.Marshal(order)
	require.NoError(t, err)

	// Sanity: provider field rendered
	assert.Contains(t, string(data), `"provider":`)

	var decoded models.Order
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, order.Reference, decoded.Reference)
	assert.Equal(t, order.ConnectorID, decoded.ConnectorID)
	assert.Equal(t, order.Direction, decoded.Direction)
	assert.Equal(t, order.Status, decoded.Status)
	assert.Equal(t, order.BaseQuantityOrdered, decoded.BaseQuantityOrdered)
	require.NotNil(t, decoded.SourceAccountID)
	assert.Equal(t, order.SourceAccountID.Reference, decoded.SourceAccountID.Reference)
	require.Len(t, decoded.Adjustments, len(order.Adjustments))
}

func TestOrderUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()
		var o models.Order
		err := json.Unmarshal([]byte("not json"), &o)
		require.Error(t, err)
	})

	t.Run("invalid id", func(t *testing.T) {
		t.Parallel()
		var o models.Order
		err := json.Unmarshal([]byte(`{"id":"!!!","connectorID":""}`), &o)
		require.Error(t, err)
	})

	t.Run("invalid connector id", func(t *testing.T) {
		t.Parallel()
		connectorID := newConnectorID(t)
		observedAt := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
		order, err := models.FromPSPOrderToOrder(validPSPOrder(), connectorID, observedAt)
		require.NoError(t, err)
		data, err := json.Marshal(order)
		require.NoError(t, err)

		// Poison the connectorID field.
		bad := []byte(
			string(data)[:0] + "{" +
				`"id":"` + order.ID.String() + `",` +
				`"connectorID":"not-a-connector-id"}`,
		)
		var decoded models.Order
		err = json.Unmarshal(bad, &decoded)
		require.Error(t, err)
	})

	t.Run("invalid source account id", func(t *testing.T) {
		t.Parallel()
		connectorID := newConnectorID(t)
		observedAt := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
		order, err := models.FromPSPOrderToOrder(validPSPOrder(), connectorID, observedAt)
		require.NoError(t, err)

		payload := map[string]any{
			"id":              order.ID.String(),
			"connectorID":     connectorID.String(),
			"sourceAccountID": "invalid-account-id",
		}
		raw, err := json.Marshal(payload)
		require.NoError(t, err)

		var decoded models.Order
		err = json.Unmarshal(raw, &decoded)
		require.Error(t, err)
	})

	t.Run("invalid destination account id", func(t *testing.T) {
		t.Parallel()
		connectorID := newConnectorID(t)
		observedAt := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
		order, err := models.FromPSPOrderToOrder(validPSPOrder(), connectorID, observedAt)
		require.NoError(t, err)

		payload := map[string]any{
			"id":                   order.ID.String(),
			"connectorID":          connectorID.String(),
			"destinationAccountID": "invalid-account-id",
		}
		raw, err := json.Marshal(payload)
		require.NoError(t, err)

		var decoded models.Order
		err = json.Unmarshal(raw, &decoded)
		require.Error(t, err)
	})
}

func TestOrderAdjustmentID(t *testing.T) {
	t.Parallel()

	orderID := models.OrderID{
		Reference:   "order-ref-1",
		ConnectorID: models.ConnectorID{Provider: "psp", Reference: uuid.New()},
	}
	original := models.OrderAdjustmentID{
		OrderID:            orderID,
		Reference:          "order-ref-1",
		Status:             models.ORDER_STATUS_FILLED,
		BaseQuantityFilled: big.NewInt(1000),
		Fee:                big.NewInt(10),
		FeeAsset:           pointer.For("USD/2"),
	}

	t.Run("String round trip", func(t *testing.T) {
		t.Parallel()
		s := original.String()
		assert.NotEmpty(t, s)

		decoded, err := models.OrderAdjustmentIDFromString(s)
		require.NoError(t, err)
		assert.Equal(t, original.Reference, decoded.Reference)
		assert.Equal(t, original.Status, decoded.Status)
	})

	t.Run("FromString invalid", func(t *testing.T) {
		t.Parallel()
		_, err := models.OrderAdjustmentIDFromString("invalid-base64!!")
		assert.Error(t, err)
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		val, err := original.Value()
		require.NoError(t, err)
		assert.Equal(t, original.String(), val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()

		t.Run("valid", func(t *testing.T) {
			t.Parallel()
			var id models.OrderAdjustmentID
			require.NoError(t, id.Scan(original.String()))
			assert.Equal(t, original.Reference, id.Reference)
		})

		t.Run("nil", func(t *testing.T) {
			t.Parallel()
			var id models.OrderAdjustmentID
			err := id.Scan(nil)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "order adjustment id is nil")
		})

		t.Run("invalid type", func(t *testing.T) {
			t.Parallel()
			var id models.OrderAdjustmentID
			err := id.Scan(123)
			assert.Error(t, err)
		})

		t.Run("illegal base64", func(t *testing.T) {
			t.Parallel()
			var id models.OrderAdjustmentID
			err := id.Scan("not-base64!!")
			assert.Error(t, err)
		})
	})
}

func TestOrderAdjustmentMarshalUnmarshal(t *testing.T) {
	t.Parallel()
	connectorID := newConnectorID(t)
	observedAt := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	adj := models.FromPSPOrderToOrderAdjustment(validPSPOrder(), connectorID, observedAt)

	data, err := json.Marshal(adj)
	require.NoError(t, err)

	var decoded models.OrderAdjustment
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, adj.Reference, decoded.Reference)
	assert.Equal(t, adj.Status, decoded.Status)
	assert.Equal(t, adj.BaseQuantityFilled, decoded.BaseQuantityFilled)
	assert.Equal(t, adj.Fee, decoded.Fee)
	assert.NotEmpty(t, decoded.IdempotencyKey())

	t.Run("invalid adjustment ID fails unmarshal", func(t *testing.T) {
		t.Parallel()
		var o models.OrderAdjustment
		err := json.Unmarshal([]byte(`{"id":"!!!bad-id"}`), &o)
		require.Error(t, err)
	})

	t.Run("malformed JSON fails", func(t *testing.T) {
		t.Parallel()
		var o models.OrderAdjustment
		err := json.Unmarshal([]byte("not json"), &o)
		require.Error(t, err)
	})
}

func TestOrderExpandedMarshal(t *testing.T) {
	t.Parallel()
	connectorID := newConnectorID(t)
	observedAt := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	order, err := models.FromPSPOrderToOrder(validPSPOrder(), connectorID, observedAt)
	require.NoError(t, err)

	t.Run("without error", func(t *testing.T) {
		t.Parallel()
		oe := models.OrderExpanded{Order: order, Status: models.ORDER_STATUS_FILLED}
		data, err := json.Marshal(oe)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"status":"FILLED"`)
		assert.NotContains(t, string(data), `"error"`)
	})

	t.Run("with error", func(t *testing.T) {
		t.Parallel()
		oe := models.OrderExpanded{
			Order:  order,
			Status: models.ORDER_STATUS_FAILED,
			Error:  errors.New("boom"),
		}
		data, err := json.Marshal(oe)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"error":"boom"`)
	})
}
