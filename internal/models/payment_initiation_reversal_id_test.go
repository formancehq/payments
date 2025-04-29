package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentInitiationReversalID(t *testing.T) {
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

		// When
		result := reversalID.String()

		// Then
		assert.NotEmpty(t, result)
	})

	t.Run("PaymentInitiationReversalIDFromString", func(t *testing.T) {
		t.Parallel()
		// Given

		original := models.PaymentInitiationReversalID{
			Reference: "rev123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}

		idStr := original.String()

		// When
		id, err := models.PaymentInitiationReversalIDFromString(idStr)

		// Then
		require.NoError(t, err)
		assert.Equal(t, original.Reference, id.Reference)
		assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)

		_, err = models.PaymentInitiationReversalIDFromString("invalid-base64")
		// Then
		assert.Error(t, err)

		_, err = models.PaymentInitiationReversalIDFromString("aW52YWxpZC1qc29u")
		// Then
		assert.Error(t, err)
	})

	t.Run("MustPaymentInitiationReversalIDFromString", func(t *testing.T) {
		t.Parallel()
		// Given

		original := models.PaymentInitiationReversalID{
			Reference: "rev123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}

		idStr := original.String()

		// When
		id := models.MustPaymentInitiationReversalIDFromString(idStr)

		// Then
		assert.Equal(t, original.Reference, id.Reference)
		assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
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

		// When
		val, err := reversalID.Value()

		// Then
		require.NoError(t, err)
		assert.Equal(t, reversalID.String(), val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		// Given

		original := models.PaymentInitiationReversalID{
			Reference: "rev123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}

		idStr := original.String()

		var id models.PaymentInitiationReversalID
		// When
		err := id.Scan(idStr)

		// Then
		require.NoError(t, err)
		assert.Equal(t, original.Reference, id.Reference)
		assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)

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
