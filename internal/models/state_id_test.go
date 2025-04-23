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
		
		id := models.StateID{
			Reference: "state123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		assert.NotEmpty(t, id.String())
	})

	t.Run("StateIDFromString", func(t *testing.T) {
		t.Parallel()
		
		original := models.StateID{
			Reference: "state123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		idStr := original.String()
		
		id, err := models.StateIDFromString(idStr)
		require.NoError(t, err)
		assert.Equal(t, original.Reference, id.Reference)
		assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
		assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())
		
		_, err = models.StateIDFromString("invalid-base64")
		assert.Error(t, err)
		
		_, err = models.StateIDFromString("aW52YWxpZC1qc29u")
		assert.Error(t, err)
	})

	t.Run("MustStateIDFromString", func(t *testing.T) {
		t.Parallel()
		
		original := models.StateID{
			Reference: "state123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		idStr := original.String()
		
		id := models.MustStateIDFromString(idStr)
		assert.Equal(t, original.Reference, id.Reference)
		assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
		assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())
		
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		
		id := models.StateID{
			Reference: "state123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		val, err := id.Value()
		require.NoError(t, err)
		assert.Equal(t, id.String(), val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		
		original := models.StateID{
			Reference: "state123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}
		
		idStr := original.String()
		
		var id models.StateID
		err := id.Scan(idStr)
		require.NoError(t, err)
		assert.Equal(t, original.Reference, id.Reference)
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
