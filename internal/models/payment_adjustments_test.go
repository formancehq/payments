package models_test

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

func TestPaymentAdjustmentIdempotencyKey(t *testing.T) {
	t.Parallel()

	paymentID := models.PaymentID{
		PaymentReference: models.PaymentReference{
			Reference: "payment123",
			Type:      models.PAYMENT_TYPE_PAYIN,
		},
		ConnectorID: models.ConnectorID{
			Provider:  "test",
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		},
	}
	
	adjustmentID := models.PaymentAdjustmentID{
		PaymentID:  paymentID,
		Reference:  "adj123",
		CreatedAt:  time.Now().UTC(),
		Status:     models.PAYMENT_STATUS_SUCCEEDED,
	}
	
	adjustment := models.PaymentAdjustment{
		ID: adjustmentID,
	}

	key := adjustment.IdempotencyKey()
	assert.Equal(t, models.IdempotencyKey(adjustmentID), key)
}

func TestPaymentAdjustmentMarshalJSON(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	paymentID := models.PaymentID{
		PaymentReference: models.PaymentReference{
			Reference: "payment123",
			Type:      models.PAYMENT_TYPE_PAYIN,
		},
		ConnectorID: models.ConnectorID{
			Provider:  "test",
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		},
	}
	
	adjustmentID := models.PaymentAdjustmentID{
		PaymentID:  paymentID,
		Reference:  "adj123",
		CreatedAt:  now,
		Status:     models.PAYMENT_STATUS_SUCCEEDED,
	}
	
	amount := big.NewInt(100)
	asset := "USD/2"
	
	adjustment := models.PaymentAdjustment{
		ID:        adjustmentID,
		Reference: "adj123",
		CreatedAt: now,
		Status:    models.PAYMENT_STATUS_SUCCEEDED,
		Amount:    amount,
		Asset:     &asset,
		Metadata: map[string]string{
			"key": "value",
		},
		Raw: json.RawMessage(`{"test": "data"}`),
	}

	data, err := json.Marshal(adjustment)
	require.NoError(t, err)

	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	require.NoError(t, err)

	assert.Equal(t, adjustmentID.String(), jsonMap["id"])
	assert.Equal(t, "adj123", jsonMap["reference"])
	assert.NotNil(t, jsonMap["createdAt"])
	assert.Equal(t, "SUCCEEDED", jsonMap["status"])
	assert.Equal(t, float64(100), jsonMap["amount"])
	assert.Equal(t, "USD/2", jsonMap["asset"])
	
	metadata, ok := jsonMap["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value", metadata["key"])
	
	raw, ok := jsonMap["raw"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "data", raw["test"])
}

func TestPaymentAdjustmentUnmarshalJSON(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	
	t.Run("valid JSON", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"id": "test:00000000-0000-0000-0000-000000000001/PAYIN/payment123/adj123/` + now.Format(time.RFC3339Nano) + `/SUCCEEDED",
			"reference": "adj123",
			"createdAt": "` + now.Format(time.RFC3339Nano) + `",
			"status": "SUCCEEDED",
			"amount": 100,
			"asset": "USD/2",
			"metadata": {"key": "value"},
			"raw": "eyJ0ZXN0IjoiZGF0YSJ9"
		}`

		var adjustment models.PaymentAdjustment
		err := json.Unmarshal([]byte(jsonData), &adjustment)
		require.NoError(t, err)

		assert.Equal(t, "adj123", adjustment.Reference)
		assert.Equal(t, now.Format(time.RFC3339), adjustment.CreatedAt.Format(time.RFC3339))
		assert.Equal(t, models.PAYMENT_STATUS_SUCCEEDED, adjustment.Status)
		assert.Equal(t, big.NewInt(100), adjustment.Amount)
		assert.Equal(t, "USD/2", *adjustment.Asset)
		assert.Equal(t, "value", adjustment.Metadata["key"])
	})

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()

		jsonData := `{invalid json}`

		var adjustment models.PaymentAdjustment
		err := json.Unmarshal([]byte(jsonData), &adjustment)
		require.Error(t, err)
	})

	t.Run("invalid ID", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"id": "invalid-id",
			"reference": "adj123",
			"createdAt": "` + now.Format(time.RFC3339Nano) + `",
			"status": "SUCCEEDED",
			"amount": 100,
			"asset": "USD/2",
			"metadata": {"key": "value"},
			"raw": "eyJ0ZXN0IjogImRhdGEifQ=="
		}`

		var adjustment models.PaymentAdjustment
		err := json.Unmarshal([]byte(jsonData), &adjustment)
		require.Error(t, err)
	})
}
