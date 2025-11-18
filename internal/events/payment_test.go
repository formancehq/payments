package events

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentMessagePayload_MarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("with amounts", func(t *testing.T) {
		t.Parallel()

		initialAmount := big.NewInt(1000000)
		amount := big.NewInt(999999)
		payload := PaymentMessagePayload{
			ID:            "pay123",
			ConnectorID:   "conn123",
			Provider:      "test",
			Reference:     "ref123",
			CreatedAt:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			Type:          "PAYIN",
			Status:        "SUCCEEDED",
			Scheme:        "SEPA",
			Asset:         "USD/2",
			InitialAmount: initialAmount,
			Amount:        amount,
		}

		data, err := json.Marshal(&payload)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "pay123", result["id"])
		assert.Equal(t, "1000000", result["initialAmount"])
		assert.Equal(t, "999999", result["amount"])
	})

	t.Run("with nil amounts", func(t *testing.T) {
		t.Parallel()

		payload := PaymentMessagePayload{
			ID:            "pay123",
			ConnectorID:   "conn123",
			Provider:      "test",
			Reference:     "ref123",
			CreatedAt:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			Type:          "PAYIN",
			Status:        "SUCCEEDED",
			Scheme:        "SEPA",
			Asset:         "USD/2",
			InitialAmount: nil,
			Amount:        nil,
		}

		data, err := json.Marshal(&payload)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Nil(t, result["initialAmount"])
		assert.Nil(t, result["amount"])
	})

	t.Run("with one nil amount", func(t *testing.T) {
		t.Parallel()

		amount := big.NewInt(500000)
		payload := PaymentMessagePayload{
			ID:            "pay123",
			ConnectorID:   "conn123",
			Provider:      "test",
			Reference:     "ref123",
			CreatedAt:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			Type:          "PAYIN",
			Status:        "SUCCEEDED",
			Scheme:        "SEPA",
			Asset:         "USD/2",
			InitialAmount: nil,
			Amount:        amount,
		}

		data, err := json.Marshal(&payload)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Nil(t, result["initialAmount"])
		assert.Equal(t, "500000", result["amount"])
	})
}

func TestPaymentMessagePayload_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("with valid amount strings", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"id": "pay123",
			"connectorID": "conn123",
			"provider": "test",
			"reference": "ref123",
			"createdAt": "2024-01-01T12:00:00Z",
			"type": "PAYIN",
			"status": "SUCCEEDED",
			"scheme": "SEPA",
			"asset": "USD/2",
			"rawData": {},
			"initialAmount": "1000000",
			"amount": "999999"
		}`

		var payload PaymentMessagePayload
		err := json.Unmarshal([]byte(jsonData), &payload)
		require.NoError(t, err)

		assert.Equal(t, "pay123", payload.ID)
		assert.NotNil(t, payload.InitialAmount)
		assert.Equal(t, "1000000", payload.InitialAmount.String())
		assert.NotNil(t, payload.Amount)
		assert.Equal(t, "999999", payload.Amount.String())
	})

	t.Run("with nil amounts", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"id": "pay123",
			"connectorID": "conn123",
			"provider": "test",
			"reference": "ref123",
			"createdAt": "2024-01-01T12:00:00Z",
			"type": "PAYIN",
			"status": "SUCCEEDED",
			"scheme": "SEPA",
			"asset": "USD/2",
			"rawData": {}
		}`

		var payload PaymentMessagePayload
		err := json.Unmarshal([]byte(jsonData), &payload)
		require.NoError(t, err)

		assert.Nil(t, payload.InitialAmount)
		assert.Nil(t, payload.Amount)
	})

	t.Run("with invalid initialAmount string", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"id": "pay123",
			"connectorID": "conn123",
			"provider": "test",
			"reference": "ref123",
			"createdAt": "2024-01-01T12:00:00Z",
			"type": "PAYIN",
			"status": "SUCCEEDED",
			"scheme": "SEPA",
			"asset": "USD/2",
			"rawData": {},
			"initialAmount": "invalid-number"
		}`

		var payload PaymentMessagePayload
		err := json.Unmarshal([]byte(jsonData), &payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid initialAmount string")
	})

	t.Run("with invalid amount string", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"id": "pay123",
			"connectorID": "conn123",
			"provider": "test",
			"reference": "ref123",
			"createdAt": "2024-01-01T12:00:00Z",
			"type": "PAYIN",
			"status": "SUCCEEDED",
			"scheme": "SEPA",
			"asset": "USD/2",
			"rawData": {},
			"amount": "invalid-number"
		}`

		var payload PaymentMessagePayload
		err := json.Unmarshal([]byte(jsonData), &payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid amount string")
	})

	t.Run("round-trip marshalling", func(t *testing.T) {
		t.Parallel()

		original := PaymentMessagePayload{
			ID:            "pay123",
			ConnectorID:   "conn123",
			Provider:      "test",
			Reference:     "ref123",
			CreatedAt:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			Type:          "PAYIN",
			Status:        "SUCCEEDED",
			Scheme:        "SEPA",
			Asset:         "USD/2",
			InitialAmount: big.NewInt(1000000),
			Amount:        big.NewInt(999999),
		}

		data, err := json.Marshal(&original)
		require.NoError(t, err)

		var unmarshaled PaymentMessagePayload
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, original.ID, unmarshaled.ID)
		assert.Equal(t, original.InitialAmount.String(), unmarshaled.InitialAmount.String())
		assert.Equal(t, original.Amount.String(), unmarshaled.Amount.String())
	})
}

