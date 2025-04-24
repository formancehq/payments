package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventID(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		// Given
		
		connectorID := &models.ConnectorID{
			Provider:  "stripe",
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}
		
		id := models.EventID{
			EventIdempotencyKey: "event123",
			ConnectorID:         connectorID,
		}
		
		// When
		result := id.String()
		
		// Then
		assert.NotEmpty(t, result)
	})

	t.Run("EventIDFromString", func(t *testing.T) {
		t.Parallel()
		// Given
		
		connectorID := &models.ConnectorID{
			Provider:  "stripe",
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}
		
		original := models.EventID{
			EventIdempotencyKey: "event123",
			ConnectorID:         connectorID,
		}
		
		idStr := original.String()
		
		// When
		id, err := models.EventIDFromString(idStr)
		
		// Then
		require.NoError(t, err)
		assert.Equal(t, original.EventIdempotencyKey, id.EventIdempotencyKey)
		assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
		assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())
		
		_, err = models.EventIDFromString("invalid-base64")
		// Then
			assert.Error(t, err)
		
		_, err = models.EventIDFromString("aW52YWxpZC1qc29u")
		// Then
			assert.Error(t, err)
	})


	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		// Given
		
		connectorID := &models.ConnectorID{
			Provider:  "stripe",
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}
		
		id := models.EventID{
			EventIdempotencyKey: "event123",
			ConnectorID:         connectorID,
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
		
		connectorID := &models.ConnectorID{
			Provider:  "stripe",
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}
		
		original := models.EventID{
			EventIdempotencyKey: "event123",
			ConnectorID:         connectorID,
		}
		
		idStr := original.String()
		
		var id models.EventID
		// When
		err := id.Scan(idStr)
		
		// Then
		require.NoError(t, err)
		assert.Equal(t, original.EventIdempotencyKey, id.EventIdempotencyKey)
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
