package connector

import (
	"errors"

	"github.com/formancehq/payments/internal/models"
)

// Plugin implementation errors.
// These are the canonical definitions used by connector plugins.
var (
	ErrNotImplemented       = errors.New("not implemented")
	ErrNotYetInstalled      = errors.New("not yet installed")
	ErrInvalidClientRequest = errors.New("invalid client request")
	ErrUpstreamRatelimit    = errors.New("rate limited by upstream server")
	ErrCurrencyNotSupported = errors.New("currency not supported")
)

// Common errors that connectors can return.
// These are aliases to internal/models for backward compatibility.
var (
	ErrInvalidConfig               = models.ErrInvalidConfig
	ErrFailedAccountCreation       = models.ErrFailedAccountCreation
	ErrMissingFromPayloadInRequest = models.ErrMissingFromPayloadInRequest
	ErrMissingAccountInRequest     = models.ErrMissingAccountInRequest
	ErrInvalidRequest              = models.ErrInvalidRequest
	ErrValidation                  = models.ErrValidation
	ErrWebhookVerification         = models.ErrWebhookVerification
	ErrMissingPageSize             = models.ErrMissingPageSize
	ErrExceededMaxPageSize         = models.ErrExceededMaxPageSize
	ErrMissingConnectorMetadata    = models.ErrMissingConnectorMetadata
	ErrMissingConnectorField       = models.ErrMissingConnectorField
)

// ConnectorValidationError represents a validation error for a specific field.
type ConnectorValidationError = models.ConnectorValidationError

// NewConnectorValidationError creates a new ConnectorValidationError.
var NewConnectorValidationError = models.NewConnectorValidationError
