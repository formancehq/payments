package models

import "errors"

var (
	ErrInvalidConfig               = errors.New("invalid config")
	ErrFailedAccountCreation       = errors.New("failed to create account")
	ErrMissingFromPayloadInRequest = errors.New("missing from payload in request")
	ErrMissingAccountInRequest     = errors.New("missing account number in request")
	ErrInvalidRequest              = errors.New("invalid request")
)
