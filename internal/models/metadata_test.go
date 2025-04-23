package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestExtractNamespacedMetadata(t *testing.T) {
	t.Parallel()

	metadata := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	extracted := models.ExtractNamespacedMetadata(metadata, "key1")
	assert.Equal(t, "value1", extracted)

	extracted = models.ExtractNamespacedMetadata(metadata, "key2")
	assert.Equal(t, "value2", extracted)

	extracted = models.ExtractNamespacedMetadata(metadata, "key3")
	assert.Equal(t, "value3", extracted)

	extracted = models.ExtractNamespacedMetadata(metadata, "nonexistent")
	assert.Empty(t, extracted)

	extracted = models.ExtractNamespacedMetadata(metadata, "")
	assert.Empty(t, extracted)

	extracted = models.ExtractNamespacedMetadata(nil, "key1")
	assert.Empty(t, extracted)
}
