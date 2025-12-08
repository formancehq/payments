package events

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentInitiationMessagePayload_MarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("with amount", func(t *testing.T) {
		t.Parallel()

		amount := big.NewInt(5000000)
		payload := PaymentInitiationMessagePayload{
			ID:          "pi123",
			ConnectorID: "conn123",
			Provider:    "test",
			Reference:   "ref123",
			CreatedAt:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			ScheduledAt: time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
			Description: "Test payment",
			Type:        "TRANSFER",
			Amount:      amount,
			Asset:       "USD/2",
		}

		data, err := json.Marshal(&payload)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "pi123", result["id"])
		assert.Equal(t, "5000000", result["amount"])
	})

	t.Run("with nil amount", func(t *testing.T) {
		t.Parallel()

		payload := PaymentInitiationMessagePayload{
			ID:          "pi123",
			ConnectorID: "conn123",
			Provider:    "test",
			Reference:   "ref123",
			CreatedAt:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			ScheduledAt: time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
			Description: "Test payment",
			Type:        "TRANSFER",
			Amount:      nil,
			Asset:       "USD/2",
		}

		data, err := json.Marshal(&payload)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Nil(t, result["amount"])
	})
}

func TestPaymentInitiationMessagePayload_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("with valid amount string", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"id": "pi123",
			"connectorID": "conn123",
			"provider": "test",
			"reference": "ref123",
			"createdAt": "2024-01-01T12:00:00Z",
			"scheduledAt": "2024-01-02T12:00:00Z",
			"description": "Test payment",
			"type": "TRANSFER",
			"amount": "5000000",
			"asset": "USD/2"
		}`

		var payload PaymentInitiationMessagePayload
		err := json.Unmarshal([]byte(jsonData), &payload)
		require.NoError(t, err)

		assert.Equal(t, "pi123", payload.ID)
		assert.NotNil(t, payload.Amount)
		assert.Equal(t, "5000000", payload.Amount.String())
	})

	t.Run("with nil amount", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"id": "pi123",
			"connectorID": "conn123",
			"provider": "test",
			"reference": "ref123",
			"createdAt": "2024-01-01T12:00:00Z",
			"scheduledAt": "2024-01-02T12:00:00Z",
			"description": "Test payment",
			"type": "TRANSFER",
			"asset": "USD/2"
		}`

		var payload PaymentInitiationMessagePayload
		err := json.Unmarshal([]byte(jsonData), &payload)
		require.NoError(t, err)

		assert.Nil(t, payload.Amount)
	})

	t.Run("with invalid amount string", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"id": "pi123",
			"connectorID": "conn123",
			"provider": "test",
			"reference": "ref123",
			"createdAt": "2024-01-01T12:00:00Z",
			"scheduledAt": "2024-01-02T12:00:00Z",
			"description": "Test payment",
			"type": "TRANSFER",
			"amount": "invalid-number",
			"asset": "USD/2"
		}`

		var payload PaymentInitiationMessagePayload
		err := json.Unmarshal([]byte(jsonData), &payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid amount string")
	})

	t.Run("round-trip marshalling", func(t *testing.T) {
		t.Parallel()

		original := PaymentInitiationMessagePayload{
			ID:          "pi123",
			ConnectorID: "conn123",
			Provider:    "test",
			Reference:   "ref123",
			CreatedAt:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			ScheduledAt: time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
			Description: "Test payment",
			Type:        "TRANSFER",
			Amount:      big.NewInt(5000000),
			Asset:       "USD/2",
		}

		data, err := json.Marshal(&original)
		require.NoError(t, err)

		var unmarshaled PaymentInitiationMessagePayload
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, original.ID, unmarshaled.ID)
		assert.Equal(t, original.Amount.String(), unmarshaled.Amount.String())
	})
}

func TestPaymentInitiationAdjustmentMessagePayload_MarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("with amount", func(t *testing.T) {
		t.Parallel()

		amount := big.NewInt(3000000)
		asset := "USD/2"
		payload := PaymentInitiationAdjustmentMessagePayload{
			ID:                  "adj123",
			PaymentInitiationID: "pi123",
			Status:              "SUCCEEDED",
			Amount:              amount,
			Asset:               &asset,
		}

		data, err := json.Marshal(&payload)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "adj123", result["id"])
		assert.Equal(t, "3000000", result["amount"])
	})

	t.Run("with nil amount", func(t *testing.T) {
		t.Parallel()

		payload := PaymentInitiationAdjustmentMessagePayload{
			ID:                  "adj123",
			PaymentInitiationID: "pi123",
			Status:              "SUCCEEDED",
			Amount:              nil,
		}

		data, err := json.Marshal(&payload)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		// When amount is nil, it should be omitted (omitempty)
		_, exists := result["amount"]
		assert.False(t, exists)
	})
}

func TestPaymentInitiationAdjustmentMessagePayload_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("with valid amount string", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"id": "adj123",
			"paymentInitiationID": "pi123",
			"status": "SUCCEEDED",
			"amount": "3000000"
		}`

		var payload PaymentInitiationAdjustmentMessagePayload
		err := json.Unmarshal([]byte(jsonData), &payload)
		require.NoError(t, err)

		assert.Equal(t, "adj123", payload.ID)
		assert.NotNil(t, payload.Amount)
		assert.Equal(t, "3000000", payload.Amount.String())
	})

	t.Run("with nil amount", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"id": "adj123",
			"paymentInitiationID": "pi123",
			"status": "SUCCEEDED"
		}`

		var payload PaymentInitiationAdjustmentMessagePayload
		err := json.Unmarshal([]byte(jsonData), &payload)
		require.NoError(t, err)

		assert.Nil(t, payload.Amount)
	})

	t.Run("with invalid amount string", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"id": "adj123",
			"paymentInitiationID": "pi123",
			"status": "SUCCEEDED",
			"amount": "invalid-number"
		}`

		var payload PaymentInitiationAdjustmentMessagePayload
		err := json.Unmarshal([]byte(jsonData), &payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid amount string")
	})

	t.Run("round-trip marshalling", func(t *testing.T) {
		t.Parallel()

		original := PaymentInitiationAdjustmentMessagePayload{
			ID:                  "adj123",
			PaymentInitiationID: "pi123",
			Status:              "SUCCEEDED",
			Amount:              big.NewInt(3000000),
		}

		data, err := json.Marshal(&original)
		require.NoError(t, err)

		var unmarshaled PaymentInitiationAdjustmentMessagePayload
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, original.ID, unmarshaled.ID)
		assert.Equal(t, original.Amount.String(), unmarshaled.Amount.String())
	})
}

