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
)

type ConnectorMetadataError struct {
	internal error
	field    string
}

func NewConnectorMetadataError(field string) *ConnectorMetadataError {
	return &ConnectorMetadataError{
		internal: fmt.Errorf("field %q is required: %w", field, ErrMissingConnectorMetadata),
		field:    field,
	}
}

func (e *ConnectorMetadataError) Error() string { return e.internal.Error() }

func (e *ConnectorMetadataError) Unwrap() error { return e.internal }
