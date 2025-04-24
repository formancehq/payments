package models_test

import (
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentInitiationReversalAdjustmentID(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		// Given
		
		reversalID := models.PaymentInitiationReversalID{
			Reference: "rev123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		adjustmentID := models.PaymentInitiationReversalAdjustmentID{
			PaymentInitiationReversalID: reversalID,
			CreatedAt:                   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
		}
		
		// When
		result := adjustmentID.String()
		
		// Then
		assert.NotEmpty(t, result)
	})

	t.Run("PaymentInitiationReversalAdjustmentIDFromString", func(t *testing.T) {
		t.Parallel()
		// Given
		
		reversalID := models.PaymentInitiationReversalID{
			Reference: "rev123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		original := models.PaymentInitiationReversalAdjustmentID{
			PaymentInitiationReversalID: reversalID,
			CreatedAt:                   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
		}
		
		idStr := original.String()
		
		// When
		id, err := models.PaymentInitiationReversalAdjustmentIDFromString(idStr)
		
		// Then
		require.NoError(t, err)
		assert.Equal(t, original.Status, id.Status)
		assert.Equal(t, original.PaymentInitiationReversalID.Reference, id.PaymentInitiationReversalID.Reference)
		
		_, err = models.PaymentInitiationReversalAdjustmentIDFromString("invalid-base64")
		// Then
			assert.Error(t, err)
		
		_, err = models.PaymentInitiationReversalAdjustmentIDFromString("aW52YWxpZC1qc29u")
		// Then
			assert.Error(t, err)
	})

	t.Run("MustPaymentInitiationReversalAdjustmentIDFromString", func(t *testing.T) {
		t.Parallel()
		// Given
		
		reversalID := models.PaymentInitiationReversalID{
			Reference: "rev123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		original := models.PaymentInitiationReversalAdjustmentID{
			PaymentInitiationReversalID: reversalID,
			CreatedAt:                   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
		}
		
		idStr := original.String()
		
		// When
		id := models.MustPaymentInitiationReversalAdjustmentIDFromString(idStr)
		
		// Then
		assert.Equal(t, original.Status, id.Status)
		assert.Equal(t, original.PaymentInitiationReversalID.Reference, id.PaymentInitiationReversalID.Reference)
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		// Given
		
		reversalID := models.PaymentInitiationReversalID{
			Reference: "rev123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		adjustmentID := models.PaymentInitiationReversalAdjustmentID{
			PaymentInitiationReversalID: reversalID,
			CreatedAt:                   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
		}
		
		// When
		val, err := adjustmentID.Value()
		
		// Then
		require.NoError(t, err)
		assert.Equal(t, adjustmentID.String(), val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		// Given
		
		reversalID := models.PaymentInitiationReversalID{
			Reference: "rev123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		original := models.PaymentInitiationReversalAdjustmentID{
			PaymentInitiationReversalID: reversalID,
			CreatedAt:                   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
		}
		
		idStr := original.String()
		
		var id models.PaymentInitiationReversalAdjustmentID
		// When
		err := id.Scan(idStr)
		
		// Then
		require.NoError(t, err)
		assert.Equal(t, original.Status, id.Status)
		assert.Equal(t, original.PaymentInitiationReversalID.Reference, id.PaymentInitiationReversalID.Reference)
		
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
