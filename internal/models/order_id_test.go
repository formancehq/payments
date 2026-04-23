package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newOrderID(t *testing.T) models.OrderID {
	t.Helper()
	return models.OrderID{
		Reference: "order123",
		ConnectorID: models.ConnectorID{
			Provider:  "stripe",
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		},
	}
}

func TestOrderID(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		id := newOrderID(t)
		result := id.String()
		assert.NotEmpty(t, result)

		decoded, err := models.OrderIDFromString(result)
		require.NoError(t, err)
		assert.Equal(t, id.Reference, decoded.Reference)
		assert.Equal(t, id.ConnectorID.Provider, decoded.ConnectorID.Provider)
		assert.Equal(t, id.ConnectorID.Reference.String(), decoded.ConnectorID.Reference.String())
	})

	t.Run("OrderIDFromString", func(t *testing.T) {
		t.Parallel()

		t.Run("valid ID", func(t *testing.T) {
			t.Parallel()
			original := newOrderID(t)
			id, err := models.OrderIDFromString(original.String())
			require.NoError(t, err)
			assert.Equal(t, original.Reference, id.Reference)
		})

		t.Run("illegal base64", func(t *testing.T) {
			t.Parallel()
			_, err := models.OrderIDFromString("invalid-format")
			assert.Error(t, err)
		})

		t.Run("empty string", func(t *testing.T) {
			t.Parallel()
			_, err := models.OrderIDFromString("")
			assert.Error(t, err)
		})
	})

	t.Run("MustOrderIDFromString", func(t *testing.T) {
		t.Parallel()

		t.Run("valid ID", func(t *testing.T) {
			t.Parallel()
			original := newOrderID(t)
			id := models.MustOrderIDFromString(original.String())
			require.NotNil(t, id)
			assert.Equal(t, original.Reference, id.Reference)
		})

		t.Run("illegal base64 panics", func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() {
				models.MustOrderIDFromString("invalid-format")
			})
		})

		t.Run("illegal json panics", func(t *testing.T) {
			t.Parallel()
			// "aW52YWxpZC1qc29u" decodes to "invalid-json" which isn't valid JSON
			assert.Panics(t, func() {
				models.MustOrderIDFromString("aW52YWxpZC1qc29u")
			})
		})
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		id := newOrderID(t)
		val, err := id.Value()
		require.NoError(t, err)
		assert.Equal(t, id.String(), val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()

		t.Run("valid string", func(t *testing.T) {
			t.Parallel()
			original := newOrderID(t)
			var id models.OrderID
			err := id.Scan(original.String())
			require.NoError(t, err)
			assert.Equal(t, original.Reference, id.Reference)
		})

		t.Run("nil value", func(t *testing.T) {
			t.Parallel()
			var id models.OrderID
			err := id.Scan(nil)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "order id is nil")
		})

		t.Run("invalid type", func(t *testing.T) {
			t.Parallel()
			var id models.OrderID
			err := id.Scan(123)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to")
		})

		t.Run("illegal base64", func(t *testing.T) {
			t.Parallel()
			var id models.OrderID
			err := id.Scan("invalid-format")
			assert.Error(t, err)
		})
	})
}
