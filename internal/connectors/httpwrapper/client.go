// Package httpwrapper re-exports pkg/connector/httpwrapper for internal use.
package httpwrapper

import (
	"github.com/formancehq/payments/pkg/connector/httpwrapper"
)

// Client is a convenience wrapper that encapsulates common code related to interacting with HTTP endpoints.
type Client = httpwrapper.Client

// NewClient creates a new HTTP client wrapper with the given configuration.
var NewClient = httpwrapper.NewClient

// Error variables re-exported from pkg/connector/httpwrapper.
var (
	ErrStatusCodeUnexpected      = httpwrapper.ErrStatusCodeUnexpected
	ErrStatusCodeClientError     = httpwrapper.ErrStatusCodeClientError
	ErrStatusCodeServerError     = httpwrapper.ErrStatusCodeServerError
	ErrStatusCodeTooManyRequests = httpwrapper.ErrStatusCodeTooManyRequests
)
