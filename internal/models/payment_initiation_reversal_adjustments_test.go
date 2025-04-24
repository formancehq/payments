package models_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentInitiationReversalAdjustmentMarshalJSON(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.New(),
	}
	reversalID := models.PaymentInitiationReversalID{
		Reference:   "rev123",
		ConnectorID: connectorID,
	}
	adjustmentID := models.PaymentInitiationReversalAdjustmentID{
		PaymentInitiationReversalID: reversalID,

		CreatedAt:                   now,
		Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
	}

	adjustment := models.PaymentInitiationReversalAdjustment{
		ID:                          adjustmentID,
		PaymentInitiationReversalID: reversalID,
		CreatedAt:                   now,
		Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
		Error:                       nil,
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
	assert.Equal(t, reversalID.String(), jsonMap["paymentInitiationReversalID"])
	assert.Equal(t, "PROCESSED", jsonMap["status"])
	assert.Equal(t, "value", jsonMap["metadata"].(map[string]interface{})["key"])
	_, hasError := jsonMap["error"]
	assert.False(t, hasError)

	testError := errors.New("test error")
	adjustment.Status = models.PAYMENT_INITIATION_REVERSAL_STATUS_FAILED
	adjustment.Error = testError

	data, err = json.Marshal(adjustment)
	// Then
			require.NoError(t, err)

	err = json.Unmarshal(data, &jsonMap)
	// Then
			require.NoError(t, err)

	assert.Equal(t, "FAILED", jsonMap["status"])
	assert.Equal(t, testError.Error(), jsonMap["error"])
}

func TestPaymentInitiationReversalAdjustmentUnmarshalJSON(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
	reversalID := models.PaymentInitiationReversalID{
		Reference:   "rev123",
		ConnectorID: connectorID,
	}
	adjustmentID := models.PaymentInitiationReversalAdjustmentID{
		PaymentInitiationReversalID: reversalID,

		CreatedAt:                   now,
		Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
	}

	originalAdjustment := models.PaymentInitiationReversalAdjustment{
		ID:                          adjustmentID,
		PaymentInitiationReversalID: reversalID,
		CreatedAt:                   now,
		Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
		Error:                       nil,
		Metadata: map[string]string{
			"key": "value",
		},
	}

	data, err := json.Marshal(originalAdjustment)
	// Then
			require.NoError(t, err)

	var adjustment models.PaymentInitiationReversalAdjustment
	err = json.Unmarshal(data, &adjustment)
	// Then
			require.NoError(t, err)

	assert.Equal(t, originalAdjustment.ID.String(), adjustment.ID.String())
	assert.Equal(t, originalAdjustment.PaymentInitiationReversalID.String(), adjustment.PaymentInitiationReversalID.String())
	assert.Equal(t, originalAdjustment.CreatedAt.Format(time.RFC3339Nano), adjustment.CreatedAt.Format(time.RFC3339Nano))
	assert.Equal(t, originalAdjustment.Status, adjustment.Status)
	assert.Nil(t, adjustment.Error)
	assert.Equal(t, originalAdjustment.Metadata["key"], adjustment.Metadata["key"])

	testError := errors.New("test error")
	originalAdjustment.Status = models.PAYMENT_INITIATION_REVERSAL_STATUS_FAILED
	originalAdjustment.Error = testError

	data, err = json.Marshal(originalAdjustment)
	// Then
			require.NoError(t, err)

	err = json.Unmarshal(data, &adjustment)
	// Then
			require.NoError(t, err)

	assert.Equal(t, models.PAYMENT_INITIATION_REVERSAL_STATUS_FAILED, adjustment.Status)
	assert.NotNil(t, adjustment.Error)
	assert.Equal(t, testError.Error(), adjustment.Error.Error())

	invalidJSON := `{
		"id": "invalid-id",
		"paymentInitiationReversalID": "test:00000000-0000-0000-0000-000000000001/rev123",
		"createdAt": "` + now.Format(time.RFC3339Nano) + `",
		"status": "PROCESSED"
	}`
	err = json.Unmarshal([]byte(invalidJSON), &adjustment)
	// Then
			assert.Error(t, err)

	invalidJSON = `{
		"id": "test:00000000-0000-0000-0000-000000000001/rev123/adj123/` + now.Format(time.RFC3339Nano) + `/PROCESSED",
		"paymentInitiationReversalID": "invalid-reversal-id",
		"createdAt": "` + now.Format(time.RFC3339Nano) + `",
		"status": "PROCESSED"
	}`
	err = json.Unmarshal([]byte(invalidJSON), &adjustment)
	// Then
			assert.Error(t, err)

	invalidJSON = `{invalid json}`
	err = json.Unmarshal([]byte(invalidJSON), &adjustment)
	// Then
			assert.Error(t, err)
}
