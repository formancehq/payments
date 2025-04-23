package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestExtractNamespacedMetadata(t *testing.T) {
	t.Parallel()

	metadata := map[string]string{
		"namespace:key1": "value1",
		"namespace:key2": "value2",
		"other:key":      "value3",
		"plain_key":      "value4",
	}

	extracted := models.ExtractNamespacedMetadata(metadata, "namespace:key1")
	assert.Equal(t, "value1", extracted)

	extracted = models.ExtractNamespacedMetadata(metadata, "namespace:key2")
	assert.Equal(t, "value2", extracted)

	extracted = models.ExtractNamespacedMetadata(metadata, "other:key")
	assert.Equal(t, "value3", extracted)

	extracted = models.ExtractNamespacedMetadata(metadata, "plain_key")
	assert.Equal(t, "value4", extracted)

	extracted = models.ExtractNamespacedMetadata(metadata, "nonexistent:key")
	assert.Empty(t, extracted)

	extracted = models.ExtractNamespacedMetadata(metadata, "")
	assert.Empty(t, extracted)

	extracted = models.ExtractNamespacedMetadata(nil, "namespace:key")
	assert.Empty(t, extracted)
}
