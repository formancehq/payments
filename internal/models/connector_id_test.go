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
		expected := "stripe:00000000-0000-0000-0000-000000000001"
		
		result := id.String()
		
		assert.Equal(t, expected, result)
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
			
			id, err := models.ConnectorIDFromString(idStr)
			
			require.NoError(t, err)
			assert.Equal(t, original.Provider, id.Provider)
			assert.Equal(t, original.Reference.String(), id.Reference.String())
		})
		
		t.Run("invalid format", func(t *testing.T) {
			t.Parallel()
			
			// Given
			invalidStr := "invalid-format"
			
			_, err := models.ConnectorIDFromString(invalidStr)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid connector ID format")
		})
		
		t.Run("invalid UUID", func(t *testing.T) {
			t.Parallel()
			
			// Given
			invalidUUID := "stripe:not-a-uuid"
			
			_, err := models.ConnectorIDFromString(invalidUUID)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid UUID")
		})
		
		t.Run("empty string", func(t *testing.T) {
			t.Parallel()
			
			// Given
			emptyStr := ""
			
			_, err := models.ConnectorIDFromString(emptyStr)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid connector ID format")
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
		
		id := models.MustConnectorIDFromString(idStr)
		
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
		
		val, err := id.Value()
		
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
			var id models.ConnectorID
			
			err := id.Scan(idStr)
			
			require.NoError(t, err)
			assert.Equal(t, original.Provider, id.Provider)
			assert.Equal(t, original.Reference.String(), id.Reference.String())
		})
		
		t.Run("invalid type", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var id models.ConnectorID
			invalidValue := 123
			
			err := id.Scan(invalidValue)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "cannot scan")
		})
		
		t.Run("invalid format", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var id models.ConnectorID
			invalidStr := "invalid-format"
			
			err := id.Scan(invalidStr)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid connector ID format")
		})
		
		t.Run("nil value", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var id models.ConnectorID
			
			err := id.Scan(nil)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "cannot scan")
		})
	})
}
