package models

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidConfig               = errors.New("invalid config")
	ErrFailedAccountCreation       = errors.New("failed to create account")
	ErrMissingFromPayloadInRequest = errors.New("missing from payload in request")
	ErrMissingAccountInRequest     = errors.New("missing account number in request")
	ErrInvalidRequest              = errors.New("invalid request")
	ErrValidation                  = errors.New("validation error")

	ErrMissingConnectorMetadata = errors.New("missing required metadata in request")
	ErrMissingConnectorField    = errors.New("missing required field in request")
)

type ConnectorValidationError struct {
	internal error
	field    string
}

var NonRetryableError *ConnectorValidationError

func NewConnectorValidationError(field string, err error) *ConnectorValidationError {
	return &ConnectorValidationError{
		internal: fmt.Errorf("validation error occurred for field %s: %w", field, err),
		field:    field,
	}
}

func (e *ConnectorValidationError) Error() string { return e.internal.Error() }

func (e *ConnectorValidationError) Unwrap() error { return e.internal }
