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

func TestPaymentInitiationAdjustmentIdempotencyKey(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.New(),
	}
	paymentInitiationID := models.PaymentInitiationID{
		Reference:   "pi123",
		ConnectorID: connectorID,
	}
	adjustmentID := models.PaymentInitiationAdjustmentID{
		PaymentInitiationID: paymentInitiationID,
		CreatedAt:           now,
		Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
	}

	adjustment := models.PaymentInitiationAdjustment{
		ID: adjustmentID,
	}

	key := adjustment.IdempotencyKey()
	assert.Equal(t, models.IdempotencyKey(adjustmentID), key)
}

func TestPaymentInitiationAdjustmentMarshalJSON(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.New(),
	}
	paymentInitiationID := models.PaymentInitiationID{
		Reference:   "pi123",
		ConnectorID: connectorID,
	}
	adjustmentID := models.PaymentInitiationAdjustmentID{
		PaymentInitiationID: paymentInitiationID,
		CreatedAt:           now,
		Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
	}

	t.Run("with no error", func(t *testing.T) {
		t.Parallel()
		// Given

		adjustment := models.PaymentInitiationAdjustment{
			ID:        adjustmentID,
			CreatedAt: now,
			Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
			Amount:    big.NewInt(100),
			Asset:     pointer.For("USD/2"),
			Error:     nil,
			Metadata: map[string]string{
				"key": "value",
			},
		}

		data, err := json.Marshal(adjustment)
		
		// Then
		require.NoError(t, err)

		var jsonMap map[string]interface{}
		err = json.Unmarshal(data, &jsonMap)
		// Then
		require.NoError(t, err)

		assert.Equal(t, adjustmentID.String(), jsonMap["id"])
		assert.Equal(t, "PROCESSED", jsonMap["status"])
		assert.Equal(t, float64(100), jsonMap["amount"])
		assert.Equal(t, "USD/2", jsonMap["asset"])
		assert.Equal(t, "value", jsonMap["metadata"].(map[string]interface{})["key"])
		_, hasError := jsonMap["error"]
		assert.False(t, hasError)
	})

	t.Run("with error", func(t *testing.T) {
		t.Parallel()
		// Given

		testError := errors.New("test error")
		adjustment := models.PaymentInitiationAdjustment{
			ID:        adjustmentID,
			CreatedAt: now,
			Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
			Amount:    big.NewInt(100),
			Asset:     pointer.For("USD/2"),
			Error:     testError,
			Metadata: map[string]string{
				"key": "value",
			},
		}

		data, err := json.Marshal(adjustment)
		
		// Then
		require.NoError(t, err)

		var jsonMap map[string]interface{}
		err = json.Unmarshal(data, &jsonMap)
		// Then
		require.NoError(t, err)

		assert.Equal(t, adjustmentID.String(), jsonMap["id"])
		assert.Equal(t, "FAILED", jsonMap["status"])
		assert.Equal(t, float64(100), jsonMap["amount"])
		assert.Equal(t, "USD/2", jsonMap["asset"])
		assert.Equal(t, "value", jsonMap["metadata"].(map[string]interface{})["key"])
		assert.Equal(t, "test error", jsonMap["error"])
	})

	t.Run("with nil amount and asset", func(t *testing.T) {
		t.Parallel()
		// Given

		adjustment := models.PaymentInitiationAdjustment{
			ID:        adjustmentID,
			CreatedAt: now,
			Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
			Amount:    nil,
			Asset:     nil,
			Error:     nil,
			Metadata: map[string]string{
				"key": "value",
			},
		}

		data, err := json.Marshal(adjustment)
		
		// Then
		require.NoError(t, err)

		var jsonMap map[string]interface{}
		err = json.Unmarshal(data, &jsonMap)
		// Then
		require.NoError(t, err)

		assert.Equal(t, adjustmentID.String(), jsonMap["id"])
		assert.Equal(t, "PROCESSED", jsonMap["status"])
		_, hasAmount := jsonMap["amount"]
		assert.False(t, hasAmount)
		_, hasAsset := jsonMap["asset"]
		assert.False(t, hasAsset)
	})
}

func TestPaymentInitiationAdjustmentUnmarshalJSON(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
	paymentInitiationID := models.PaymentInitiationID{
		Reference:   "pi123",
		ConnectorID: connectorID,
	}
	adjustmentID := models.PaymentInitiationAdjustmentID{
		PaymentInitiationID: paymentInitiationID,
		CreatedAt:           now,
		Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
	}

	t.Run("valid adjustment with no error", func(t *testing.T) {
		t.Parallel()
		// Given

		jsonData := `{
			"id": "` + adjustmentID.String() + `",
			"createdAt": "` + now.Format(time.RFC3339Nano) + `",
			"status": "PROCESSED",
			"amount": 100,
			"asset": "USD/2",
			"metadata": {
				"key": "value"
			}
		}`

		var adjustment models.PaymentInitiationAdjustment
		err := json.Unmarshal([]byte(jsonData), &adjustment)
		
		// Then
		require.NoError(t, err)

		assert.Equal(t, adjustmentID.String(), adjustment.ID.String())
		assert.Equal(t, now.Format(time.RFC3339Nano), adjustment.CreatedAt.Format(time.RFC3339Nano))
		assert.Equal(t, models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED, adjustment.Status)
		assert.Equal(t, big.NewInt(100), adjustment.Amount)
		assert.Equal(t, "USD/2", *adjustment.Asset)
		assert.Nil(t, adjustment.Error)
		assert.Equal(t, "value", adjustment.Metadata["key"])
	})

	t.Run("valid adjustment with error", func(t *testing.T) {
		t.Parallel()
		// Given

		jsonData := `{
			"id": "` + adjustmentID.String() + `",
			"createdAt": "` + now.Format(time.RFC3339Nano) + `",
			"status": "FAILED",
			"amount": 100,
			"asset": "USD/2",
			"error": "test error",
			"metadata": {
				"key": "value"
			}
		}`

		var adjustment models.PaymentInitiationAdjustment
		err := json.Unmarshal([]byte(jsonData), &adjustment)
		
		// Then
		require.NoError(t, err)

		assert.Equal(t, adjustmentID.String(), adjustment.ID.String())
		assert.Equal(t, now.Format(time.RFC3339Nano), adjustment.CreatedAt.Format(time.RFC3339Nano))
		assert.Equal(t, models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED, adjustment.Status)
		assert.Equal(t, big.NewInt(100), adjustment.Amount)
		assert.Equal(t, "USD/2", *adjustment.Asset)
		assert.NotNil(t, adjustment.Error)
		assert.Equal(t, "test error", adjustment.Error.Error())
		assert.Equal(t, "value", adjustment.Metadata["key"])
	})

	t.Run("invalid ID", func(t *testing.T) {
		t.Parallel()
		// Given

		jsonData := `{
			"id": "invalid-id",
			"createdAt": "` + now.Format(time.RFC3339Nano) + `",
			"status": "PROCESSED",
			"metadata": {}
		}`

		var adjustment models.PaymentInitiationAdjustment
		err := json.Unmarshal([]byte(jsonData), &adjustment)
		
		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "illegal base64")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()
		// Given

		jsonData := `{invalid json}`

		var adjustment models.PaymentInitiationAdjustment
		err := json.Unmarshal([]byte(jsonData), &adjustment)
		
		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})
}
