package models_test

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentInitiationIdempotencyKey(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.New(),
	}

	paymentInitiation := models.PaymentInitiation{
		ID: models.PaymentInitiationID{
			Reference:   "pi123",
			ConnectorID: connectorID,

		},
		ConnectorID:  connectorID,
		Reference:    "pi123",
		CreatedAt:    now,
		ScheduledAt:  now.Add(time.Hour),
		Description:  "Test payment initiation",
		Type:         models.PAYMENT_INITIATION_TYPE_TRANSFER,
		Amount:       big.NewInt(100),
		Asset:        "USD/2",
		Metadata:     map[string]string{"key": "value"},
	}

	key := paymentInitiation.IdempotencyKey()
	assert.NotEmpty(t, key)

	key2 := paymentInitiation.IdempotencyKey()
	assert.Equal(t, key, key2)

	paymentInitiation2 := models.PaymentInitiation{
		ID: models.PaymentInitiationID{
			Reference:   "pi456",
			ConnectorID: connectorID,

		},
	}
	key3 := paymentInitiation2.IdempotencyKey()
	assert.NotEqual(t, key, key3)
}

func TestPaymentInitiationMarshalJSON(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.New(),
	}
	sourceAccountID := models.AccountID{
		Reference:   "source123",
		ConnectorID: connectorID,
	}
	destinationAccountID := models.AccountID{
		Reference:   "dest123",
		ConnectorID: connectorID,
	}

	paymentInitiation := models.PaymentInitiation{
		ID: models.PaymentInitiationID{
			Reference:   "pi123",
			ConnectorID: connectorID,

		},
		ConnectorID:         connectorID,
		Reference:           "pi123",
		CreatedAt:           now,
		ScheduledAt:         now.Add(time.Hour),
		Description:         "Test payment initiation",
		Type:                models.PAYMENT_INITIATION_TYPE_TRANSFER,
		SourceAccountID:     &sourceAccountID,
		DestinationAccountID: &destinationAccountID,
		Amount:              big.NewInt(100),
		Asset:               "USD/2",
		Metadata:            map[string]string{"key": "value"},
	}

	data, err := json.Marshal(paymentInitiation)
	require.NoError(t, err)

	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	require.NoError(t, err)

	assert.Equal(t, paymentInitiation.ID.String(), jsonMap["id"])
	assert.Equal(t, connectorID.String(), jsonMap["connectorID"])
	assert.Equal(t, "test", jsonMap["provider"])
	assert.Equal(t, "pi123", jsonMap["reference"])
	assert.Equal(t, "Test payment initiation", jsonMap["description"])
	assert.Equal(t, "TRANSFER", jsonMap["paymentInitiationType"])
	assert.Equal(t, sourceAccountID.String(), jsonMap["sourceAccountID"])
	assert.Equal(t, destinationAccountID.String(), jsonMap["destinationAccountID"])
	assert.Equal(t, float64(100), jsonMap["amount"])
	assert.Equal(t, "USD/2", jsonMap["asset"])
	assert.Equal(t, "value", jsonMap["metadata"].(map[string]interface{})["key"])

	paymentInitiation.SourceAccountID = nil
	paymentInitiation.DestinationAccountID = nil

	data, err = json.Marshal(paymentInitiation)
	require.NoError(t, err)

	err = json.Unmarshal(data, &jsonMap)
	require.NoError(t, err)

	sourceAccountIDValue, hasSourceAccountID := jsonMap["sourceAccountID"]
	assert.True(t, hasSourceAccountID)
	assert.NotNil(t, sourceAccountIDValue)
	assert.IsType(t, "", sourceAccountIDValue)
	destinationAccountIDValue, hasDestinationAccountID := jsonMap["destinationAccountID"]
	assert.True(t, hasDestinationAccountID)
	assert.NotNil(t, destinationAccountIDValue)
	assert.IsType(t, "", destinationAccountIDValue)
}

func TestPaymentInitiationUnmarshalJSON(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
	sourceAccountID := models.AccountID{
		Reference:   "source123",
		ConnectorID: connectorID,
	}
	destinationAccountID := models.AccountID{
		Reference:   "dest123",
		ConnectorID: connectorID,
	}

	originalPI := models.PaymentInitiation{
		ID: models.PaymentInitiationID{
			Reference:   "pi123",
			ConnectorID: connectorID,

		},
		ConnectorID:         connectorID,
		Reference:           "pi123",
		CreatedAt:           now,
		ScheduledAt:         now.Add(time.Hour),
		Description:         "Test payment initiation",
		Type:                models.PAYMENT_INITIATION_TYPE_TRANSFER,
		SourceAccountID:     &sourceAccountID,
		DestinationAccountID: &destinationAccountID,
		Amount:              big.NewInt(100),
		Asset:               "USD/2",
		Metadata:            map[string]string{"key": "value"},
	}

	data, err := json.Marshal(originalPI)
	require.NoError(t, err)

	var pi models.PaymentInitiation
	err = json.Unmarshal(data, &pi)
	require.NoError(t, err)

	assert.Equal(t, originalPI.ID.String(), pi.ID.String())
	assert.Equal(t, originalPI.ConnectorID.String(), pi.ConnectorID.String())
	assert.Equal(t, originalPI.Reference, pi.Reference)
	assert.Equal(t, originalPI.CreatedAt.Format(time.RFC3339Nano), pi.CreatedAt.Format(time.RFC3339Nano))
	assert.Equal(t, originalPI.ScheduledAt.Format(time.RFC3339Nano), pi.ScheduledAt.Format(time.RFC3339Nano))
	assert.Equal(t, originalPI.Description, pi.Description)
	assert.Equal(t, originalPI.Type, pi.Type)
	assert.Equal(t, originalPI.SourceAccountID.String(), pi.SourceAccountID.String())
	assert.Equal(t, originalPI.DestinationAccountID.String(), pi.DestinationAccountID.String())
	assert.Equal(t, originalPI.Amount.String(), pi.Amount.String())
	assert.Equal(t, originalPI.Asset, pi.Asset)
	assert.Equal(t, originalPI.Metadata["key"], pi.Metadata["key"])

	invalidJSON := `{
		"id": "invalid-id",
		"connectorID": "test:00000000-0000-0000-0000-000000000001",
		"reference": "pi123"
	}`
	err = json.Unmarshal([]byte(invalidJSON), &pi)
	assert.Error(t, err)

	invalidJSON = `{
		"id": "test:00000000-0000-0000-0000-000000000001/TRANSFER/pi123",
		"connectorID": "invalid-connector",
		"reference": "pi123"
	}`
	err = json.Unmarshal([]byte(invalidJSON), &pi)
	assert.Error(t, err)

	invalidJSON = `{
		"id": "test:00000000-0000-0000-0000-000000000001/TRANSFER/pi123",
		"connectorID": "test:00000000-0000-0000-0000-000000000001",
		"reference": "pi123",
		"sourceAccountID": "invalid-account"
	}`
	err = json.Unmarshal([]byte(invalidJSON), &pi)
	assert.Error(t, err)

	invalidJSON = `{
		"id": "test:00000000-0000-0000-0000-000000000001/TRANSFER/pi123",
		"connectorID": "test:00000000-0000-0000-0000-000000000001",
		"reference": "pi123",
		"destinationAccountID": "invalid-account"
	}`
	err = json.Unmarshal([]byte(invalidJSON), &pi)
	assert.Error(t, err)
}

func TestFromPaymentInitiationToPSPPaymentInitiation(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.New(),
	}
	sourceAccountID := models.AccountID{
		Reference:   "source123",
		ConnectorID: connectorID,
	}
	destinationAccountID := models.AccountID{
		Reference:   "dest123",
		ConnectorID: connectorID,
	}

	sourceAccount := &models.PSPAccount{
		Reference: "source123",
		CreatedAt: now,
		Name:      pointer.For("Source Account"),
	}

	destinationAccount := &models.PSPAccount{
		Reference: "dest123",
		CreatedAt: now,
		Name:      pointer.For("Destination Account"),
	}

	paymentInitiation := models.PaymentInitiation{
		ID: models.PaymentInitiationID{
			Reference:   "pi123",
			ConnectorID: connectorID,

		},
		ConnectorID:         connectorID,
		Reference:           "pi123",
		CreatedAt:           now,
		ScheduledAt:         now.Add(time.Hour),
		Description:         "Test payment initiation",
		Type:                models.PAYMENT_INITIATION_TYPE_TRANSFER,
		SourceAccountID:     &sourceAccountID,
		DestinationAccountID: &destinationAccountID,
		Amount:              big.NewInt(100),
		Asset:               "USD/2",
		Metadata:            map[string]string{"key": "value"},
	}

	pspPI := models.FromPaymentInitiationToPSPPaymentInitiation(&paymentInitiation, sourceAccount, destinationAccount)

	assert.Equal(t, "pi123", pspPI.Reference)
	assert.Equal(t, now.Add(time.Hour).Format(time.RFC3339Nano), pspPI.CreatedAt.Format(time.RFC3339Nano))
	assert.Equal(t, "Test payment initiation", pspPI.Description)
	assert.Equal(t, sourceAccount, pspPI.SourceAccount)
	assert.Equal(t, destinationAccount, pspPI.DestinationAccount)
	assert.Equal(t, big.NewInt(100), pspPI.Amount)
	assert.Equal(t, "USD/2", pspPI.Asset)
	assert.Equal(t, "value", pspPI.Metadata["key"])

	paymentInitiation.ScheduledAt = time.Time{}
	pspPI = models.FromPaymentInitiationToPSPPaymentInitiation(&paymentInitiation, sourceAccount, destinationAccount)
	assert.Equal(t, now.Format(time.RFC3339Nano), pspPI.CreatedAt.Format(time.RFC3339Nano))
}

func TestPaymentInitiationExpandedMarshalJSON(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.New(),
	}
	sourceAccountID := models.AccountID{
		Reference:   "source123",
		ConnectorID: connectorID,
	}
	destinationAccountID := models.AccountID{
		Reference:   "dest123",
		ConnectorID: connectorID,
	}

	paymentInitiation := models.PaymentInitiation{
		ID: models.PaymentInitiationID{
			Reference:   "pi123",
			ConnectorID: connectorID,

		},
		ConnectorID:         connectorID,
		Reference:           "pi123",
		CreatedAt:           now,
		ScheduledAt:         now.Add(time.Hour),
		Description:         "Test payment initiation",
		Type:                models.PAYMENT_INITIATION_TYPE_TRANSFER,
		SourceAccountID:     &sourceAccountID,
		DestinationAccountID: &destinationAccountID,
		Amount:              big.NewInt(100),
		Asset:               "USD/2",
		Metadata:            map[string]string{"key": "value"},
	}

	expanded := models.PaymentInitiationExpanded{
		PaymentInitiation: paymentInitiation,
		Status:            models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
		Error:             nil,
	}

	data, err := json.Marshal(expanded)
	require.NoError(t, err)

	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	require.NoError(t, err)

	assert.Equal(t, paymentInitiation.ID.String(), jsonMap["id"])
	assert.Equal(t, connectorID.String(), jsonMap["connectorID"])
	assert.Equal(t, "test", jsonMap["provider"])
	assert.Equal(t, "pi123", jsonMap["reference"])
	assert.Equal(t, "Test payment initiation", jsonMap["description"])
	assert.Equal(t, "TRANSFER", jsonMap["type"])
	assert.Equal(t, sourceAccountID.String(), jsonMap["sourceAccountID"])
	assert.Equal(t, destinationAccountID.String(), jsonMap["destinationAccountID"])
	assert.Equal(t, float64(100), jsonMap["amount"])
	assert.Equal(t, "USD/2", jsonMap["asset"])
	assert.Equal(t, "value", jsonMap["metadata"].(map[string]interface{})["key"])
	assert.Equal(t, "PROCESSED", jsonMap["status"])
	_, hasError := jsonMap["error"]
	assert.False(t, hasError)

	expanded = models.PaymentInitiationExpanded{
		PaymentInitiation: paymentInitiation,
		Status:            models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
		Error:             assert.AnError,
	}

	data, err = json.Marshal(expanded)
	require.NoError(t, err)

	err = json.Unmarshal(data, &jsonMap)
	require.NoError(t, err)

	assert.Equal(t, "FAILED", jsonMap["status"])
	assert.Equal(t, assert.AnError.Error(), jsonMap["error"])

	paymentInitiation.SourceAccountID = nil
	paymentInitiation.DestinationAccountID = nil
	expanded = models.PaymentInitiationExpanded{
		PaymentInitiation: paymentInitiation,
		Status:            models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
		Error:             nil,
	}

	data, err = json.Marshal(expanded)
	require.NoError(t, err)

	err = json.Unmarshal(data, &jsonMap)
	require.NoError(t, err)

	sourceAccountIDValue, hasSourceAccountID := jsonMap["sourceAccountID"]
	assert.True(t, hasSourceAccountID)
	assert.NotNil(t, sourceAccountIDValue)
	assert.IsType(t, "", sourceAccountIDValue)
	destinationAccountIDValue, hasDestinationAccountID := jsonMap["destinationAccountID"]
	assert.True(t, hasDestinationAccountID)
	assert.NotNil(t, destinationAccountIDValue)
	assert.IsType(t, "", destinationAccountIDValue)
}
