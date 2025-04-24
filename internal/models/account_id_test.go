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
		
		// When/Then
		assert.NotEmpty(t, id.String())
	})

	t.Run("AccountIDFromString", func(t *testing.T) {
		t.Parallel()
		// Given
		
		original := models.AccountID{
			Reference:   "acc123",
			ConnectorID: models.ConnectorID{Provider: "stripe", Reference: uuid.New()},
		}
		
		idStr := original.String()
		
		id, err := models.AccountIDFromString(idStr)
		// When/Then
		require.NoError(t, err)
		assert.Equal(t, original.Reference, id.Reference)
		assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
		assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())
		
		_, err = models.AccountIDFromString("invalid-format")
		assert.Error(t, err)
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
		// When/Then
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
		// When/Then
		require.NoError(t, err)
		assert.Equal(t, id.String(), val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		// Given
		
		original := models.AccountID{
			Reference:   "acc123",
			ConnectorID: models.ConnectorID{Provider: "stripe", Reference: uuid.New()},
		}
		
		idStr := original.String()
		
		var id models.AccountID
		err := id.Scan(idStr)
		// When/Then
		require.NoError(t, err)
		assert.Equal(t, original.Reference, id.Reference)
		
		var id3 models.AccountID
		err = id3.Scan(123)
		assert.Error(t, err)
		
		var id4 models.AccountID
		err = id4.Scan("invalid-format")
		assert.Error(t, err)
	})
}
