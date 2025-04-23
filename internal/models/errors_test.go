package models_test

import (
	"errors"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestConnectorMetadataError(t *testing.T) {
	t.Parallel()
	
	expectedField := "arbitrary-field-name"
	err := models.NewConnectorValidationError(expectedField, models.ErrMissingConnectorMetadata)
	assert.Regexp(t, expectedField, err.Error())
	assert.ErrorIs(t, err, models.ErrMissingConnectorMetadata)
}

func TestConnectorValidationError(t *testing.T) {
	t.Parallel()

	originalErr := errors.New("test error")
	validationErr := models.NewConnectorValidationError("test_field", originalErr)
	
	assert.Contains(t, validationErr.Error(), "test_field")
	assert.Contains(t, validationErr.Error(), "test error")
	
	unwrappedErr := validationErr.Unwrap()
	assert.NotNil(t, unwrappedErr)
	assert.Contains(t, unwrappedErr.Error(), "test_field")
	assert.Contains(t, unwrappedErr.Error(), "test error")
}

func TestErrorVariables(t *testing.T) {
	t.Parallel()
	
	assert.NotNil(t, models.ErrInvalidConfig)
	assert.NotNil(t, models.ErrFailedAccountCreation)
	assert.NotNil(t, models.ErrMissingFromPayloadInRequest)
	assert.NotNil(t, models.ErrMissingAccountInRequest)
	assert.NotNil(t, models.ErrInvalidRequest)
	assert.NotNil(t, models.ErrValidation)
	assert.NotNil(t, models.ErrMissingConnectorMetadata)
	assert.NotNil(t, models.ErrMissingConnectorField)
	
	assert.Equal(t, "invalid config", models.ErrInvalidConfig.Error())
	assert.Equal(t, "failed to create account", models.ErrFailedAccountCreation.Error())
	assert.Equal(t, "missing from payload in request", models.ErrMissingFromPayloadInRequest.Error())
	assert.Equal(t, "missing account number in request", models.ErrMissingAccountInRequest.Error())
	assert.Equal(t, "invalid request", models.ErrInvalidRequest.Error())
	assert.Equal(t, "validation error", models.ErrValidation.Error())
	assert.Equal(t, "missing required metadata in request", models.ErrMissingConnectorMetadata.Error())
	assert.Equal(t, "missing required field in request", models.ErrMissingConnectorField.Error())
}

func TestNonRetryableError(t *testing.T) {
	t.Parallel()
	
	assert.NotNil(t, models.NonRetryableError)
}
