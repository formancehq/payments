package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestConnectorMetadataError(t *testing.T) {
	expectedField := "arbitrary-field-name"
	err := models.NewConnectorMetadataError(expectedField)
	assert.Regexp(t, expectedField, err.Error())
	assert.ErrorIs(t, err, models.ErrMissingConnectorMetadata)
}
