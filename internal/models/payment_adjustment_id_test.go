package models_test

import (
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentAdjustmentID(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		// Given
		
		paymentID := models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: "payment123",
				Type:      models.PAYMENT_TYPE_PAYIN,
			},
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		adjustmentID := models.PaymentAdjustmentID{
			PaymentID: paymentID,
			Reference: "adj123",
			CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:    models.PAYMENT_STATUS_SUCCEEDED,
		}
		
		// When/Then
		assert.NotEmpty(t, adjustmentID.String())
	})

	t.Run("PaymentAdjustmentIDFromString", func(t *testing.T) {
		t.Parallel()
		// Given
		
		paymentID := models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: "payment123",
				Type:      models.PAYMENT_TYPE_PAYIN,
			},
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		original := models.PaymentAdjustmentID{
			PaymentID: paymentID,
			Reference: "adj123",
			CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:    models.PAYMENT_STATUS_SUCCEEDED,
		}
		
		idStr := original.String()
		
		// When
			id, err := models.PaymentAdjustmentIDFromString(idStr)
		// When/Then
		// Then
			require.NoError(t, err)
		assert.Equal(t, original.Reference, id.Reference)
		assert.Equal(t, original.Status, id.Status)
		assert.Equal(t, original.PaymentID.PaymentReference.Reference, id.PaymentID.PaymentReference.Reference)
		
		_, err = models.PaymentAdjustmentIDFromString("invalid-base64")
		// Then
			assert.Error(t, err)
		
		_, err = models.PaymentAdjustmentIDFromString("aW52YWxpZC1qc29u")
		// Then
			assert.Error(t, err)
	})

	t.Run("MustPaymentAdjustmentIDFromString", func(t *testing.T) {
		t.Parallel()
		// Given
		
		paymentID := models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: "payment123",
				Type:      models.PAYMENT_TYPE_PAYIN,
			},
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		original := models.PaymentAdjustmentID{
			PaymentID: paymentID,
			Reference: "adj123",
			CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:    models.PAYMENT_STATUS_SUCCEEDED,
		}
		
		idStr := original.String()
		
		id := models.MustPaymentAdjustmentIDFromString(idStr)
		// When/Then
		assert.Equal(t, original.Reference, id.Reference)
		assert.Equal(t, original.Status, id.Status)
		assert.Equal(t, original.PaymentID.PaymentReference.Reference, id.PaymentID.PaymentReference.Reference)
		
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		// Given
		
		paymentID := models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: "payment123",
				Type:      models.PAYMENT_TYPE_PAYIN,
			},
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		adjustmentID := models.PaymentAdjustmentID{
			PaymentID: paymentID,
			Reference: "adj123",
			CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:    models.PAYMENT_STATUS_SUCCEEDED,
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
		
		paymentID := models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: "payment123",
				Type:      models.PAYMENT_TYPE_PAYIN,
			},
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		original := models.PaymentAdjustmentID{
			PaymentID: paymentID,
			Reference: "adj123",
			CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:    models.PAYMENT_STATUS_SUCCEEDED,
		}
		
		idStr := original.String()
		
		var id models.PaymentAdjustmentID
		// When
			err := id.Scan(idStr)
		// When/Then
		// Then
			require.NoError(t, err)
		assert.Equal(t, original.Reference, id.Reference)
		assert.Equal(t, original.Status, id.Status)
		assert.Equal(t, original.PaymentID.PaymentReference.Reference, id.PaymentID.PaymentReference.Reference)
		
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
