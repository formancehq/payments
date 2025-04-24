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
		
		result := id.String()
		
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "stripe:")
		assert.Contains(t, result, "/acc123")
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
			
			id, err := models.AccountIDFromString(idStr)
			
			require.NoError(t, err)
			assert.Equal(t, original.Reference, id.Reference)
			assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
			assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())
		})
		
		t.Run("invalid format", func(t *testing.T) {
			t.Parallel()
			
			// Given
			invalidStr := "invalid-format"
			
			_, err := models.AccountIDFromString(invalidStr)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid account ID format")
		})
		
		t.Run("empty string", func(t *testing.T) {
			t.Parallel()
			
			// Given
			emptyStr := ""
			
			_, err := models.AccountIDFromString(emptyStr)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid account ID format")
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
		
		id := models.MustAccountIDFromString(idStr)
		
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
		
		val, err := id.Value()
		
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
			var accountID models.AccountID
			
			err := accountID.Scan(idStr)
			
			require.NoError(t, err)
			assert.Equal(t, original.Reference, accountID.Reference)
		})
		
		t.Run("invalid type", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var accountID models.AccountID
			invalidValue := 123
			
			err := accountID.Scan(invalidValue)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "cannot scan")
		})
		
		t.Run("invalid format", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var accountID models.AccountID
			invalidStr := "invalid-format"
			
			err := accountID.Scan(invalidStr)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid account ID format")
		})
		
		t.Run("nil value", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var accountID models.AccountID
			
			err := accountID.Scan(nil)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "cannot scan")
		})
	})
}
