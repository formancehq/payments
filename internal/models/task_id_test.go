package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskID(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		// Given

		id := models.TaskID{
			Reference: "task123",
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

	t.Run("TaskIDFromString", func(t *testing.T) {
		t.Parallel()
		// Given

		original := models.TaskID{
			Reference: "task123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}

		idStr := original.String()

		// When
		id, err := models.TaskIDFromString(idStr)

		// Then
		require.NoError(t, err)
		assert.Equal(t, original.Reference, id.Reference)
		assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
		assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())

		_, err = models.TaskIDFromString("invalid-base64")
		// Then
		assert.Error(t, err)

		_, err = models.TaskIDFromString("aW52YWxpZC1qc29u")
		// Then
		assert.Error(t, err)
	})

	t.Run("MustTaskIDFromString", func(t *testing.T) {
		t.Parallel()
		// Given

		original := models.TaskID{
			Reference: "task123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}

		idStr := original.String()

		// When
		id := models.MustTaskIDFromString(idStr)

		// Then
		assert.Equal(t, original.Reference, id.Reference)
		assert.Equal(t, original.ConnectorID.Provider, id.ConnectorID.Provider)
		assert.Equal(t, original.ConnectorID.Reference.String(), id.ConnectorID.Reference.String())
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		// Given

		id := models.TaskID{
			Reference: "task123",
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

		original := models.TaskID{
			Reference: "task123",
			ConnectorID: models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
		}

		idStr := original.String()

		var id models.TaskID
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
