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
		assert.NotEmpty(t, id.String())
	})

	t.Run("PaymentIDFromString", func(t *testing.T) {
		t.Parallel()
		
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
		
		_, err = models.PaymentIDFromString("invalid-format")
		assert.Error(t, err)
	})

	t.Run("MustPaymentIDFromString", func(t *testing.T) {
		t.Parallel()
		
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
		
		
		var id3 models.PaymentID
		err = id3.Scan(123)
		assert.Error(t, err)
		
		var id4 models.PaymentID
		err = id4.Scan("invalid-format")
		assert.Error(t, err)
	})
}
