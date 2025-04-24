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
		// When/Then
		assert.NotEmpty(t, id.String())
	})

	t.Run("ConnectorIDFromString", func(t *testing.T) {
		t.Parallel()
		// Given
		
		original := models.ConnectorID{
			Provider:  "stripe",
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}
		
		idStr := original.String()
		
		id, err := models.ConnectorIDFromString(idStr)
		// When/Then
		require.NoError(t, err)
		assert.Equal(t, original.Provider, id.Provider)
		assert.Equal(t, original.Reference.String(), id.Reference.String())
		
		_, err = models.ConnectorIDFromString("invalid-format")
		assert.Error(t, err)
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
		// When/Then
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
		// When/Then
		require.NoError(t, err)
		assert.Equal(t, id.String(), val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		// Given
		
		original := models.ConnectorID{
			Provider:  "stripe",
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}
		
		idStr := original.String()
		
		var id models.ConnectorID
		err := id.Scan(idStr)
		// When/Then
		require.NoError(t, err)
		assert.Equal(t, original.Provider, id.Provider)
		assert.Equal(t, original.Reference.String(), id.Reference.String())
		
		var id3 models.ConnectorID
		err = id3.Scan(123)
		assert.Error(t, err)
		
		var id4 models.ConnectorID
		err = id4.Scan("invalid-format")
		assert.Error(t, err)
	})
}
