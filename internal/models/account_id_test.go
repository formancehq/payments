package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountID(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		// Given
		id := models.AccountID{
			Reference:   "acc123",
			ConnectorID: models.ConnectorID{Provider: "stripe", Reference: uuid.New()},
		}

		// When
		result := id.String()

		// Then
		assert.NotEmpty(t, result)
		decoded, err := models.AccountIDFromString(result)
		require.NoError(t, err)
		assert.Equal(t, id.Reference, decoded.Reference)
		assert.Equal(t, id.ConnectorID.Provider, decoded.ConnectorID.Provider)
	})

	t.Run("AccountIDFromString", func(t *testing.T) {
		t.Parallel()

		t.Run("valid ID", func(t *testing.T) {
			t.Parallel()

			// Given
			original := models.AccountID{
				Reference:   "acc123",
				ConnectorID: models.ConnectorID{Provider: "stripe", Reference: uuid.New()},
			}
			idStr := original.String()

			// When
			id, err := models.AccountIDFromString(idStr)

			// Then
			require.NoError(t, err)
			assert.Equal(t, original.Reference, id.Reference)
			assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
			assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())
		})

		t.Run("invalid format", func(t *testing.T) {
			t.Parallel()

			// Given
			invalidStr := "invalid-format"

			// When
			_, err := models.AccountIDFromString(invalidStr)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid character")
		})

		t.Run("empty string", func(t *testing.T) {
			t.Parallel()

			// Given
			emptyStr := ""

			// When
			_, err := models.AccountIDFromString(emptyStr)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unexpected end of JSON input")
		})
	})

	t.Run("MustAccountIDFromString", func(t *testing.T) {
		t.Parallel()

		// Given
		original := models.AccountID{
			Reference:   "acc123",
			ConnectorID: models.ConnectorID{Provider: "stripe", Reference: uuid.New()},
		}
		idStr := original.String()

		// When
		id := models.MustAccountIDFromString(idStr)

		// Then
		assert.Equal(t, original.Reference, id.Reference)
		assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
		assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()

		// Given
		id := models.AccountID{
			Reference:   "acc123",
			ConnectorID: models.ConnectorID{Provider: "stripe", Reference: uuid.New()},
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
			original := models.AccountID{
				Reference:   "acc123",
				ConnectorID: models.ConnectorID{Provider: "stripe", Reference: uuid.New()},
			}
			idStr := original.String()
			var id1 models.AccountID

			// When
			err := id1.Scan(idStr)

			// Then
			require.NoError(t, err)
			assert.Equal(t, original.Reference, id1.Reference)
		})

		t.Run("invalid type", func(t *testing.T) {
			t.Parallel()

			// Given
			var id2 models.AccountID
			invalidValue := 123

			// When
			err := id2.Scan(invalidValue)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse account id")
		})

		t.Run("invalid format", func(t *testing.T) {
			t.Parallel()

			// Given
			var id3 models.AccountID
			invalidStr := "invalid-format"

			// When
			err := id3.Scan(invalidStr)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse account id")
			assert.Contains(t, err.Error(), "invalid character")
		})

		t.Run("nil value", func(t *testing.T) {
			t.Parallel()

			// Given
			var id4 models.AccountID

			// When
			err := id4.Scan(nil)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "account id is nil")
		})
	})
}
