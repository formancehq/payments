package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateID(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		// Given
		
		id := models.StateID{
			Reference: "state123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		// When
		result := id.String()
		
		// Then
		assert.NotEmpty(t, result)
	})

	t.Run("StateIDFromString", func(t *testing.T) {
		t.Parallel()
		// Given
		
		original := models.StateID{
			Reference: "state123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		idStr := original.String()
		
		// When
		id, err := models.StateIDFromString(idStr)
		
		// Then
		require.NoError(t, err)
		assert.Equal(t, original.Reference, id.Reference)
		assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
		assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())
		
		_, err = models.StateIDFromString("invalid-base64")
		// Then
			assert.Error(t, err)
		
		_, err = models.StateIDFromString("aW52YWxpZC1qc29u")
		// Then
			assert.Error(t, err)
	})

	t.Run("MustStateIDFromString", func(t *testing.T) {
		t.Parallel()
		// Given
		
		original := models.StateID{
			Reference: "state123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		idStr := original.String()
		
		// When
		id := models.MustStateIDFromString(idStr)
		
		// Then
		assert.Equal(t, original.Reference, id.Reference)
		assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
		assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())
		
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		// Given
		
		id := models.StateID{
			Reference: "state123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		// When
		val, err := id.Value()
		
		// Then
		require.NoError(t, err)
		assert.Equal(t, id.String(), val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		// Given
		
		original := models.StateID{
			Reference: "state123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		idStr := original.String()
		
		var id models.StateID
		// When
		err := id.Scan(idStr)
		
		// Then
		require.NoError(t, err)
		assert.Equal(t, original.Reference, id.Reference)
		assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
		assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())
		
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
