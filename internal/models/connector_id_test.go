package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectorID(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		// Given
		id := models.ConnectorID{
			Provider:  "stripe",
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}

		// When
		result := id.String()

		// Then
		assert.NotEmpty(t, result)
		decoded, err := models.ConnectorIDFromString(result)
		require.NoError(t, err)
		assert.Equal(t, id.Provider, decoded.Provider)
		assert.Equal(t, id.Reference, decoded.Reference)
	})

	t.Run("ConnectorIDFromString", func(t *testing.T) {
		t.Parallel()

		t.Run("valid ID", func(t *testing.T) {
			t.Parallel()

			// Given
			original := models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			}
			idStr := original.String()

			// When
			id, err := models.ConnectorIDFromString(idStr)

			// Then
			require.NoError(t, err)
			assert.Equal(t, original.Provider, id.Provider)
			assert.Equal(t, original.Reference.String(), id.Reference.String())
		})

		t.Run("illegal base64", func(t *testing.T) {
			t.Parallel()

			// Given
			invalidStr := "invalid-format"

			// When
			_, err := models.ConnectorIDFromString(invalidStr)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid character")
		})

		t.Run("invalid JSON", func(t *testing.T) {
			t.Parallel()

			// Given
			invalidJSON := "eyJQcm92aWRlciI6InN0cmlwZSJ9" // Base64 of {"Provider":"stripe"} - missing Reference field

			// When
			id, err := models.ConnectorIDFromString(invalidJSON)

			// Then
			assert.Equal(t, models.ConnectorID{Provider: "stripe"}, id)
			assert.NoError(t, err)
		})

		t.Run("empty string", func(t *testing.T) {
			t.Parallel()

			// Given
			emptyStr := ""

			// When
			_, err := models.ConnectorIDFromString(emptyStr)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "JSON input")
		})
	})

	t.Run("MustConnectorIDFromString", func(t *testing.T) {
		t.Parallel()

		// Given
		original := models.ConnectorID{
			Provider:  "stripe",
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}
		idStr := original.String()

		// When
		id := models.MustConnectorIDFromString(idStr)

		// Then
		assert.Equal(t, original.Provider, id.Provider)
		assert.Equal(t, original.Reference.String(), id.Reference.String())
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()

		// Given
		id := models.ConnectorID{
			Provider:  "stripe",
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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
			original := models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			}
			idStr := original.String()
			var id1 models.ConnectorID

			// When
			err := id1.Scan(idStr)

			// Then
			require.NoError(t, err)
			assert.Equal(t, original.Provider, id1.Provider)
			assert.Equal(t, original.Reference.String(), id1.Reference.String())
		})

		t.Run("invalid type", func(t *testing.T) {
			t.Parallel()

			// Given
			var id2 models.ConnectorID
			invalidValue := 123

			// When
			err := id2.Scan(invalidValue)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse connector id")
		})

		t.Run("illegal base64", func(t *testing.T) {
			t.Parallel()

			// Given
			var id3 models.ConnectorID
			invalidStr := "invalid-format"

			// When
			err := id3.Scan(invalidStr)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse connector id")
		})

		t.Run("nil value", func(t *testing.T) {
			t.Parallel()

			// Given
			var id4 models.ConnectorID

			// When
			err := id4.Scan(nil)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "connector id is nil")
		})
	})
}
