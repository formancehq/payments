package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentID(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		
		// Given
		id := models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: "payment123",
				Type:      models.PAYMENT_TYPE_PAYIN,
			},
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		expected := "stripe:00000000-0000-0000-0000-000000000001/PAYIN/payment123"
		
		result := id.String()
		
		assert.Equal(t, expected, result)
	})

	t.Run("PaymentIDFromString", func(t *testing.T) {
		t.Parallel()
		
		t.Run("valid ID", func(t *testing.T) {
			t.Parallel()
			
			// Given
			original := models.PaymentID{
				PaymentReference: models.PaymentReference{
					Reference: "payment123",
					Type:      models.PAYMENT_TYPE_PAYIN,
				},
				ConnectorID: models.ConnectorID{
					Provider:  "stripe",
					Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			}
			idStr := original.String()
			
			id, err := models.PaymentIDFromString(idStr)
			
			require.NoError(t, err)
			assert.Equal(t, original.PaymentReference.Reference, id.PaymentReference.Reference)
			assert.Equal(t, original.PaymentReference.Type, id.PaymentReference.Type)
			assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
			assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())
		})
		
		t.Run("invalid format", func(t *testing.T) {
			t.Parallel()
			
			// Given
			invalidStr := "invalid-format"
			
			_, err := models.PaymentIDFromString(invalidStr)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid payment ID format")
		})
		
		t.Run("invalid payment type", func(t *testing.T) {
			t.Parallel()
			
			// Given
			invalidType := "stripe:00000000-0000-0000-0000-000000000001/INVALID/payment123"
			
			_, err := models.PaymentIDFromString(invalidType)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid payment type")
		})
		
		t.Run("empty string", func(t *testing.T) {
			t.Parallel()
			
			// Given
			emptyStr := ""
			
			_, err := models.PaymentIDFromString(emptyStr)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid payment ID format")
		})
	})

	t.Run("MustPaymentIDFromString", func(t *testing.T) {
		t.Parallel()
		
		// Given
		original := models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: "payment123",
				Type:      models.PAYMENT_TYPE_PAYIN,
			},
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		idStr := original.String()
		
		id := models.MustPaymentIDFromString(idStr)
		
		assert.Equal(t, original.PaymentReference.Reference, id.PaymentReference.Reference)
		assert.Equal(t, original.PaymentReference.Type, id.PaymentReference.Type)
		assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
		assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		
		// Given
		id := models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: "payment123",
				Type:      models.PAYMENT_TYPE_PAYIN,
			},
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		val, err := id.Value()
		
		require.NoError(t, err)
		assert.Equal(t, id.String(), val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		
		t.Run("valid string", func(t *testing.T) {
			t.Parallel()
			
			// Given
			original := models.PaymentID{
				PaymentReference: models.PaymentReference{
					Reference: "payment123",
					Type:      models.PAYMENT_TYPE_PAYIN,
				},
				ConnectorID: models.ConnectorID{
					Provider:  "stripe",
					Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			}
			idStr := original.String()
			var id models.PaymentID
			
			err := id.Scan(idStr)
			
			require.NoError(t, err)
			assert.Equal(t, original.PaymentReference.Reference, id.PaymentReference.Reference)
			assert.Equal(t, original.PaymentReference.Type, id.PaymentReference.Type)
		})
		
		t.Run("invalid type", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var id models.PaymentID
			invalidValue := 123
			
			err := id.Scan(invalidValue)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "cannot scan")
		})
		
		t.Run("invalid format", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var id models.PaymentID
			invalidStr := "invalid-format"
			
			err := id.Scan(invalidStr)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid payment ID format")
		})
		
		t.Run("nil value", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var id models.PaymentID
			
			err := id.Scan(nil)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "cannot scan")
		})
	})
}
