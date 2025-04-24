package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTaskIDReference(t *testing.T) {
	t.Parallel()

	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}

	t.Run("empty objectID", func(t *testing.T) {
		t.Parallel()
		// Given
		
		reference := models.TaskIDReference("prefix", connectorID, "")
		expected := "prefix-test-00000000-0000-0000-0000-000000000001"
		
		result := reference
		
		assert.Equal(t, expected, result)
	})

	t.Run("non-empty objectID", func(t *testing.T) {
		t.Parallel()
		// Given
		
		reference := models.TaskIDReference("prefix", connectorID, "object123")
		expected := "prefix-test-00000000-0000-0000-0000-000000000001-object123"
		// When/Then
		assert.Equal(t, expected, reference)
	})
}
