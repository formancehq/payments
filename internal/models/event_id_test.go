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
		
		connectorID := &models.ConnectorID{
			Provider:  "stripe",
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}
		
		id := models.EventID{
			EventIdempotencyKey: "event123",
			ConnectorID:         connectorID,
		}
		
		assert.NotEmpty(t, id.String())
	})

	t.Run("EventIDFromString", func(t *testing.T) {
		t.Parallel()
		
		connectorID := &models.ConnectorID{
			Provider:  "stripe",
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}
		
		original := models.EventID{
			EventIdempotencyKey: "event123",
			ConnectorID:         connectorID,
		}
		
		idStr := original.String()
		
		id, err := models.EventIDFromString(idStr)
		require.NoError(t, err)
		assert.Equal(t, original.EventIdempotencyKey, id.EventIdempotencyKey)
		assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
		assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())
		
		_, err = models.EventIDFromString("invalid-base64")
		assert.Error(t, err)
		
		_, err = models.EventIDFromString("aW52YWxpZC1qc29u")
		assert.Error(t, err)
	})


	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		
		connectorID := &models.ConnectorID{
			Provider:  "stripe",
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}
		
		id := models.EventID{
			EventIdempotencyKey: "event123",
			ConnectorID:         connectorID,
		}
		
		val, err := id.Value()
		require.NoError(t, err)
		assert.Equal(t, id.String(), val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		
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
		err := id.Scan(idStr)
		require.NoError(t, err)
		assert.Equal(t, original.EventIdempotencyKey, id.EventIdempotencyKey)
		assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
		assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())
		
		err = id.Scan(nil)
		assert.Error(t, err)
		
		err = id.Scan(123)
		assert.Error(t, err)
		
		err = id.Scan("invalid-base64")
		assert.Error(t, err)
	})
}
