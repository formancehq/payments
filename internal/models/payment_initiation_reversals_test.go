package models_test

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentInitiationReversalMarshalJSON(t *testing.T) {
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
	reversalID := models.PaymentInitiationReversalID{
		Reference:   "rev123",
		ConnectorID: connectorID,
	}

	reversal := models.PaymentInitiationReversal{
		ID:                  reversalID,
		ConnectorID:         connectorID,
		PaymentInitiationID: paymentInitiationID,
		Reference:           "rev123",
		CreatedAt:           now,
		Description:         "Test reversal",
		Amount:              big.NewInt(100),
		Asset:               "USD/2",
		Metadata: map[string]string{
			"key": "value",
		},
	}

	data, err := json.Marshal(reversal)
	// Then
			require.NoError(t, err)

	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	// Then
			require.NoError(t, err)

	assert.Equal(t, reversalID.String(), jsonMap["id"])
	assert.Equal(t, connectorID.String(), jsonMap["connectorID"])
	assert.Equal(t, paymentInitiationID.String(), jsonMap["paymentInitiationID"])
	assert.Equal(t, "rev123", jsonMap["reference"])
	assert.Equal(t, "Test reversal", jsonMap["description"])
	assert.Equal(t, "USD/2", jsonMap["asset"])
	assert.Equal(t, "value", jsonMap["metadata"].(map[string]interface{})["key"])
}

func TestPaymentInitiationReversalUnmarshalJSON(t *testing.T) {
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
	reversalID := models.PaymentInitiationReversalID{
		Reference:   "rev123",
		ConnectorID: connectorID,
	}

	originalReversal := models.PaymentInitiationReversal{
		ID:                  reversalID,
		ConnectorID:         connectorID,
		PaymentInitiationID: paymentInitiationID,
		Reference:           "rev123",
		CreatedAt:           now,
		Description:         "Test reversal",
		Amount:              big.NewInt(100),
		Asset:               "USD/2",
		Metadata: map[string]string{
			"key": "value",
		},
	}

	data, err := json.Marshal(originalReversal)
	// Then
			require.NoError(t, err)

	var reversal models.PaymentInitiationReversal
	err = json.Unmarshal(data, &reversal)
	// Then
			require.NoError(t, err)

	assert.Equal(t, originalReversal.ID.String(), reversal.ID.String())
	assert.Equal(t, originalReversal.ConnectorID.String(), reversal.ConnectorID.String())
	assert.Equal(t, originalReversal.PaymentInitiationID.String(), reversal.PaymentInitiationID.String())
	assert.Equal(t, originalReversal.Reference, reversal.Reference)
	assert.Equal(t, originalReversal.CreatedAt.Format(time.RFC3339Nano), reversal.CreatedAt.Format(time.RFC3339Nano))
	assert.Equal(t, originalReversal.Description, reversal.Description)
	assert.Equal(t, originalReversal.Amount, reversal.Amount)
	assert.Equal(t, originalReversal.Asset, reversal.Asset)
	assert.Equal(t, originalReversal.Metadata["key"], reversal.Metadata["key"])

	invalidJSON := `{
		"id": "invalid-id",
		"connectorID": "test:00000000-0000-0000-0000-000000000001",
		"paymentInitiationID": "test:00000000-0000-0000-0000-000000000001/TRANSFER/pi123",
		"reference": "rev123"
	}`
	err = json.Unmarshal([]byte(invalidJSON), &reversal)
	// Then
			assert.Error(t, err)

	invalidJSON = `{
		"id": "` + reversalID.String() + `",
		"connectorID": "invalid-connector",
		"paymentInitiationID": "` + paymentInitiationID.String() + `",
		"reference": "rev123"
	}`
	err = json.Unmarshal([]byte(invalidJSON), &reversal)
	// Then
			assert.Error(t, err)

	invalidJSON = `{
		"id": "` + reversalID.String() + `",
		"connectorID": "` + connectorID.String() + `",
		"paymentInitiationID": "invalid-payment-initiation",
		"reference": "rev123"
	}`
	err = json.Unmarshal([]byte(invalidJSON), &reversal)
	// Then
			assert.Error(t, err)
}

func TestFromPaymentInitiationReversalToPSPPaymentInitiationReversal(t *testing.T) {
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
	reversalID := models.PaymentInitiationReversalID{
		Reference:   "rev123",
		ConnectorID: connectorID,
	}

	reversal := models.PaymentInitiationReversal{
		ID:                  reversalID,
		ConnectorID:         connectorID,
		PaymentInitiationID: paymentInitiationID,
		Reference:           "rev123",
		CreatedAt:           now,
		Description:         "Test reversal",
		Amount:              big.NewInt(100),
		Asset:               "USD/2",
		Metadata: map[string]string{
			"key": "value",
		},
	}

	relatedPI := models.PSPPaymentInitiation{
		Reference: "pi123",
		CreatedAt: now,
		Amount:    big.NewInt(1000),
		Asset:     "USD/2",
	}

	pspReversal := models.FromPaymentInitiationReversalToPSPPaymentInitiationReversal(&reversal, relatedPI)

	assert.Equal(t, "rev123", pspReversal.Reference)
	assert.Equal(t, now, pspReversal.CreatedAt)
	assert.Equal(t, "Test reversal", pspReversal.Description)
	assert.Equal(t, relatedPI, pspReversal.RelatedPaymentInitiation)
	assert.Equal(t, big.NewInt(100), pspReversal.Amount)
	assert.Equal(t, "USD/2", pspReversal.Asset)
	assert.Equal(t, "value", pspReversal.Metadata["key"])
}

func TestPaymentInitiationReversalExpandedMarshalJSON(t *testing.T) {
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
	reversalID := models.PaymentInitiationReversalID{
		Reference:   "rev123",
		ConnectorID: connectorID,
	}

	reversal := models.PaymentInitiationReversal{
		ID:                  reversalID,
		ConnectorID:         connectorID,
		PaymentInitiationID: paymentInitiationID,
		Reference:           "rev123",
		CreatedAt:           now,
		Description:         "Test reversal",
		Amount:              big.NewInt(100),
		Asset:               "USD/2",
		Metadata: map[string]string{
			"key": "value",
		},
	}

	expanded := models.PaymentInitiationReversalExpanded{
		PaymentInitiationReversal: reversal,
		Status:                    models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
		Error:                     nil,
	}

	data, err := json.Marshal(expanded)
	// Then
			require.NoError(t, err)

	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	// Then
			require.NoError(t, err)

	assert.Equal(t, reversalID.String(), jsonMap["id"])
	assert.Equal(t, connectorID.String(), jsonMap["connectorID"])
	assert.Equal(t, paymentInitiationID.String(), jsonMap["paymentInitiationID"])
	assert.Equal(t, "rev123", jsonMap["reference"])
	assert.Equal(t, "Test reversal", jsonMap["description"])
	assert.Equal(t, "USD/2", jsonMap["asset"])
	assert.Equal(t, "value", jsonMap["metadata"].(map[string]interface{})["key"])
	assert.Equal(t, "PROCESSED", jsonMap["status"])
	assert.Nil(t, jsonMap["error"])

	expanded = models.PaymentInitiationReversalExpanded{
		PaymentInitiationReversal: reversal,
		Status:                    models.PAYMENT_INITIATION_REVERSAL_STATUS_FAILED,
		Error:                     errors.New("test error"),
	}

	data, err = json.Marshal(expanded)
	// Then
			require.NoError(t, err)

	err = json.Unmarshal(data, &jsonMap)
	// Then
			require.NoError(t, err)

	assert.Equal(t, "FAILED", jsonMap["status"])
	assert.Equal(t, "test error", jsonMap["error"])
}
