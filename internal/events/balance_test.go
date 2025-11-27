package events

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBalanceMessagePayload_MarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("with balance", func(t *testing.T) {
		t.Parallel()

		balance := big.NewInt(123456789)
		payload := BalanceMessagePayload{
			AccountID:     "acc123",
			ConnectorID:   "conn123",
			Provider:      "test",
			CreatedAt:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			LastUpdatedAt: time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
			Asset:         "USD/2",
			Balance:       balance,
		}

		data, err := json.Marshal(&payload)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "acc123", result["accountID"])
		assert.Equal(t, "conn123", result["connectorID"])
		assert.Equal(t, "test", result["provider"])
		assert.Equal(t, "USD/2", result["asset"])
		assert.Equal(t, "123456789", result["balance"])
	})

	t.Run("with nil balance", func(t *testing.T) {
		t.Parallel()

		payload := BalanceMessagePayload{
			AccountID:     "acc123",
			ConnectorID:   "conn123",
			Provider:      "test",
			CreatedAt:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			LastUpdatedAt: time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
			Asset:         "USD/2",
			Balance:       nil,
		}

		data, err := json.Marshal(&payload)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Nil(t, result["balance"])
	})

	t.Run("with zero balance", func(t *testing.T) {
		t.Parallel()

		balance := big.NewInt(0)
		payload := BalanceMessagePayload{
			AccountID:     "acc123",
			ConnectorID:   "conn123",
			Provider:      "test",
			CreatedAt:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			LastUpdatedAt: time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
			Asset:         "USD/2",
			Balance:       balance,
		}

		data, err := json.Marshal(&payload)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "0", result["balance"])
	})
}

func TestBalanceMessagePayload_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("with valid balance string", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"accountID": "acc123",
			"connectorID": "conn123",
			"provider": "test",
			"createdAt": "2024-01-01T12:00:00Z",
			"lastUpdatedAt": "2024-01-02T12:00:00Z",
			"asset": "USD/2",
			"balance": "123456789"
		}`

		var payload BalanceMessagePayload
		err := json.Unmarshal([]byte(jsonData), &payload)
		require.NoError(t, err)

		assert.Equal(t, "acc123", payload.AccountID)
		assert.Equal(t, "conn123", payload.ConnectorID)
		assert.Equal(t, "test", payload.Provider)
		assert.Equal(t, "USD/2", payload.Asset)
		assert.NotNil(t, payload.Balance)
		assert.Equal(t, "123456789", payload.Balance.String())
	})

	t.Run("with nil balance", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"accountID": "acc123",
			"connectorID": "conn123",
			"provider": "test",
			"createdAt": "2024-01-01T12:00:00Z",
			"lastUpdatedAt": "2024-01-02T12:00:00Z",
			"asset": "USD/2"
		}`

		var payload BalanceMessagePayload
		err := json.Unmarshal([]byte(jsonData), &payload)
		require.NoError(t, err)

		assert.Nil(t, payload.Balance)
	})

	t.Run("with invalid balance string", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"accountID": "acc123",
			"connectorID": "conn123",
			"provider": "test",
			"createdAt": "2024-01-01T12:00:00Z",
			"lastUpdatedAt": "2024-01-02T12:00:00Z",
			"asset": "USD/2",
			"balance": "invalid-number"
		}`

		var payload BalanceMessagePayload
		err := json.Unmarshal([]byte(jsonData), &payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid balance string")
	})

	t.Run("round-trip marshalling", func(t *testing.T) {
		t.Parallel()

		original := BalanceMessagePayload{
			AccountID:     "acc123",
			ConnectorID:   "conn123",
			Provider:      "test",
			CreatedAt:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			LastUpdatedAt: time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
			Asset:         "USD/2",
			Balance:       big.NewInt(987654321),
		}

		data, err := json.Marshal(&original)
		require.NoError(t, err)

		var unmarshaled BalanceMessagePayload
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, original.AccountID, unmarshaled.AccountID)
		assert.Equal(t, original.ConnectorID, unmarshaled.ConnectorID)
		assert.Equal(t, original.Provider, unmarshaled.Provider)
		assert.Equal(t, original.Asset, unmarshaled.Asset)
		assert.Equal(t, original.Balance.String(), unmarshaled.Balance.String())
	})
}

