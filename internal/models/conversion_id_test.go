package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newConversionID(t *testing.T) models.ConversionID {
	t.Helper()
	return models.ConversionID{
		Reference: "conv123",
		ConnectorID: models.ConnectorID{
			Provider:  "coinbaseprime",
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		},
	}
}

func TestConversionID(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		id := newConversionID(t)
		result := id.String()
		assert.NotEmpty(t, result)

		decoded, err := models.ConversionIDFromString(result)
		require.NoError(t, err)
		assert.Equal(t, id.Reference, decoded.Reference)
		assert.Equal(t, id.ConnectorID.Provider, decoded.ConnectorID.Provider)
	})

	t.Run("ConversionIDFromString", func(t *testing.T) {
		t.Parallel()

		t.Run("valid", func(t *testing.T) {
			t.Parallel()
			original := newConversionID(t)
			id, err := models.ConversionIDFromString(original.String())
			require.NoError(t, err)
			assert.Equal(t, original.Reference, id.Reference)
		})

		t.Run("illegal base64", func(t *testing.T) {
			t.Parallel()
			_, err := models.ConversionIDFromString("invalid-format")
			assert.Error(t, err)
		})

		t.Run("empty string", func(t *testing.T) {
			t.Parallel()
			_, err := models.ConversionIDFromString("")
			assert.Error(t, err)
		})
	})

	t.Run("MustConversionIDFromString", func(t *testing.T) {
		t.Parallel()

		t.Run("valid", func(t *testing.T) {
			t.Parallel()
			original := newConversionID(t)
			id := models.MustConversionIDFromString(original.String())
			require.NotNil(t, id)
			assert.Equal(t, original.Reference, id.Reference)
		})

		t.Run("illegal base64 panics", func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() {
				models.MustConversionIDFromString("invalid-format")
			})
		})

		t.Run("illegal json panics", func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() {
				models.MustConversionIDFromString("aW52YWxpZC1qc29u")
			})
		})
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		id := newConversionID(t)
		val, err := id.Value()
		require.NoError(t, err)
		assert.Equal(t, id.String(), val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()

		t.Run("valid", func(t *testing.T) {
			t.Parallel()
			original := newConversionID(t)
			var id models.ConversionID
			err := id.Scan(original.String())
			require.NoError(t, err)
			assert.Equal(t, original.Reference, id.Reference)
		})

		t.Run("nil", func(t *testing.T) {
			t.Parallel()
			var id models.ConversionID
			err := id.Scan(nil)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "conversion id is nil")
		})

		t.Run("invalid type", func(t *testing.T) {
			t.Parallel()
			var id models.ConversionID
			err := id.Scan(123)
			assert.Error(t, err)
		})

		t.Run("illegal base64", func(t *testing.T) {
			t.Parallel()
			var id models.ConversionID
			err := id.Scan("invalid-format")
			assert.Error(t, err)
		})
	})
}
