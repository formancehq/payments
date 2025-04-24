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
		
		// When
		result := id.String()
		
		// Then
		assert.NotEmpty(t, result)
		decoded, err := models.PaymentIDFromString(result)
		require.NoError(t, err)
		assert.Equal(t, id.PaymentReference.Reference, decoded.PaymentReference.Reference)
		assert.Equal(t, id.PaymentReference.Type, decoded.PaymentReference.Type)
		assert.Equal(t, id.ConnectorID.Provider, decoded.ConnectorID.Provider)
		assert.Equal(t, id.ConnectorID.Reference.String(), decoded.ConnectorID.Reference.String())
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
			
			// When
			id, err := models.PaymentIDFromString(idStr)
			
			// Then
			require.NoError(t, err)
			assert.Equal(t, original.PaymentReference.Reference, id.PaymentReference.Reference)
			assert.Equal(t, original.PaymentReference.Type, id.PaymentReference.Type)
			assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
			assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())
		})
		
		t.Run("illegal base64", func(t *testing.T) {
			t.Parallel()
			
			// Given
			invalidStr := "invalid-format"
			
			// When
			_, err := models.PaymentIDFromString(invalidStr)
			
			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid character")
		})
		
		t.Run("invalid payment type", func(t *testing.T) {
			t.Parallel()
			
			// Given
			original := models.PaymentID{
				PaymentReference: models.PaymentReference{
					Reference: "payment123",
					Type:      models.PAYMENT_TYPE_UNKNOWN,
				},
				ConnectorID: models.ConnectorID{
					Provider:  "stripe",
					Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			}
			idStr := original.String()
			
			// When
			id, err := models.PaymentIDFromString(idStr)
			
			// Then
			require.NoError(t, err)
			assert.Equal(t, models.PAYMENT_TYPE_UNKNOWN, id.PaymentReference.Type)
		})
		
		t.Run("empty string", func(t *testing.T) {
			t.Parallel()
			
			// Given
			emptyStr := ""
			
			// When
			_, err := models.PaymentIDFromString(emptyStr)
			
			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "JSON input")
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
		
		// When
		id := models.MustPaymentIDFromString(idStr)
		
		// Then
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
		
		// When
		val, err := id.Value()
		
		// Then
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
			var id1 models.PaymentID
			
			// When
			err := id1.Scan(idStr)
			
			// Then
			require.NoError(t, err)
			assert.Equal(t, original.PaymentReference.Reference, id1.PaymentReference.Reference)
			assert.Equal(t, original.PaymentReference.Type, id1.PaymentReference.Type)
		})
		
		t.Run("invalid type", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var id2 models.PaymentID
			invalidValue := 123
			
			// When
			err := id2.Scan(invalidValue)
			
			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse paymentid")
		})
		
		t.Run("illegal base64", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var id3 models.PaymentID
			invalidStr := "invalid-format"
			
			// When
			err := id3.Scan(invalidStr)
			
			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse paymentid")
		})
		
		t.Run("nil value", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var id4 models.PaymentID
			
			// When
			err := id4.Scan(nil)
			
			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "payment id is nil")
		})
	})
}
