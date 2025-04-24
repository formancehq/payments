package models_test

import (
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentInitiationAdjustmentID(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		// Given
		
		initiationID := models.PaymentInitiationID{
			Reference: "init123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		adjustmentID := models.PaymentInitiationAdjustmentID{
			PaymentInitiationID: initiationID,
			CreatedAt:           time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
		}
		
		// When/Then
		assert.NotEmpty(t, adjustmentID.String())
	})

	t.Run("PaymentInitiationAdjustmentIDFromString", func(t *testing.T) {
		t.Parallel()
		// Given
		
		initiationID := models.PaymentInitiationID{
			Reference: "init123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		original := models.PaymentInitiationAdjustmentID{
			PaymentInitiationID: initiationID,
			CreatedAt:           time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
		}
		
		idStr := original.String()
		
		// When
			id, err := models.PaymentInitiationAdjustmentIDFromString(idStr)
		// When/Then
		// Then
			require.NoError(t, err)
		assert.Equal(t, original.Status, id.Status)
		assert.Equal(t, original.PaymentInitiationID.Reference, id.PaymentInitiationID.Reference)
		
		_, err = models.PaymentInitiationAdjustmentIDFromString("invalid-base64")
		// Then
			assert.Error(t, err)
		
		_, err = models.PaymentInitiationAdjustmentIDFromString("aW52YWxpZC1qc29u")
		// Then
			assert.Error(t, err)
	})

	t.Run("MustPaymentInitiationAdjustmentIDFromString", func(t *testing.T) {
		t.Parallel()
		// Given
		
		initiationID := models.PaymentInitiationID{
			Reference: "init123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		original := models.PaymentInitiationAdjustmentID{
			PaymentInitiationID: initiationID,
			CreatedAt:           time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
		}
		
		idStr := original.String()
		
		id := models.MustPaymentInitiationAdjustmentIDFromString(idStr)
		// When/Then
		assert.Equal(t, original.Status, id.Status)
		assert.Equal(t, original.PaymentInitiationID.Reference, id.PaymentInitiationID.Reference)
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		// Given
		
		initiationID := models.PaymentInitiationID{
			Reference: "init123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		adjustmentID := models.PaymentInitiationAdjustmentID{
			PaymentInitiationID: initiationID,
			CreatedAt:           time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
		}
		
		val, err := adjustmentID.Value()
		// When/Then
		// Then
			require.NoError(t, err)
		assert.Equal(t, adjustmentID.String(), val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		// Given
		
		initiationID := models.PaymentInitiationID{
			Reference: "init123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		original := models.PaymentInitiationAdjustmentID{
			PaymentInitiationID: initiationID,
			CreatedAt:           time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
		}
		
		idStr := original.String()
		
		var id models.PaymentInitiationAdjustmentID
		// When
			err := id.Scan(idStr)
		// When/Then
		// Then
			require.NoError(t, err)
		assert.Equal(t, original.Status, id.Status)
		assert.Equal(t, original.PaymentInitiationID.Reference, id.PaymentInitiationID.Reference)
		
		err = id.Scan(nil)
		// Then
			assert.Error(t, err)
		
		err = id.Scan(123)
		// Then
			assert.Error(t, err)
		
		err = id.Scan("invalid-base64")
		// Then
			assert.Error(t, err)
	})
}
