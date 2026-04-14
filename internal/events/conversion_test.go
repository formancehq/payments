package events

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConversionMessagePayload_MarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("with all big.Int fields", func(t *testing.T) {
		t.Parallel()

		feeAsset := "USD/2"
		payload := ConversionMessagePayload{
			ID:                "conv-1",
			ConnectorID:       "conn-1",
			Provider:          "coinbaseprime",
			Reference:         "ref-1",
			CreatedAt:         time.Date(2026, 2, 9, 15, 33, 0, 0, time.UTC),
			UpdatedAt:         time.Date(2026, 2, 9, 15, 34, 0, 0, time.UTC),
			SourceAsset:       "USD/2",
			DestinationAsset:  "USDC/6",
			SourceAmount:      big.NewInt(10000),
			DestinationAmount: big.NewInt(10000000),
			Fee:               big.NewInt(50),
			FeeAsset:          &feeAsset,
			Status:            "COMPLETED",
		}

		data, err := json.Marshal(&payload)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "conv-1", result["id"])
		assert.Equal(t, "10000", result["sourceAmount"])
		assert.Equal(t, "10000000", result["destinationAmount"])
		assert.Equal(t, "50", result["fee"])
		assert.Equal(t, "COMPLETED", result["status"])
	})

	t.Run("with nil optional fields", func(t *testing.T) {
		t.Parallel()

		payload := ConversionMessagePayload{
			ID:           "conv-2",
			SourceAsset:  "USD/2",
			SourceAmount: big.NewInt(200),
			Status:       "PENDING",
		}

		data, err := json.Marshal(&payload)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "200", result["sourceAmount"])
		assert.Nil(t, result["destinationAmount"])
		assert.Nil(t, result["fee"])
	})

	t.Run("round-trip", func(t *testing.T) {
		t.Parallel()

		feeAsset := "USD/2"
		original := ConversionMessagePayload{
			ID:                   "conv-rt",
			ConnectorID:          "conn-1",
			Provider:             "coinbaseprime",
			Reference:            "ref-rt",
			SourceAsset:          "USD/2",
			DestinationAsset:     "USDC/6",
			SourceAmount:         big.NewInt(100000),
			DestinationAmount:    big.NewInt(100000000),
			Fee:                  big.NewInt(0),
			FeeAsset:             &feeAsset,
			Status:               "COMPLETED",
			SourceAccountID:      "eyJhY2N0MSJ9",
			DestinationAccountID: "eyJhY2N0MiJ9",
			Metadata:             map[string]string{"key": "value"},
			Raw:                  json.RawMessage(`{"raw":"data"}`),
		}

		data, err := json.Marshal(&original)
		require.NoError(t, err)

		var restored ConversionMessagePayload
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		assert.Equal(t, original.ID, restored.ID)
		assert.Equal(t, original.SourceAsset, restored.SourceAsset)
		assert.Equal(t, original.DestinationAsset, restored.DestinationAsset)
		assert.Equal(t, 0, original.SourceAmount.Cmp(restored.SourceAmount))
		assert.Equal(t, 0, original.DestinationAmount.Cmp(restored.DestinationAmount))
		assert.Equal(t, 0, original.Fee.Cmp(restored.Fee))
		assert.Equal(t, *original.FeeAsset, *restored.FeeAsset)
		assert.Equal(t, original.SourceAccountID, restored.SourceAccountID)
		assert.Equal(t, original.DestinationAccountID, restored.DestinationAccountID)
		assert.Equal(t, original.Metadata["key"], restored.Metadata["key"])
	})
}
