package plugins

import (
	"github.com/formancehq/payments/pkg/connector"
)

// Error aliases for backward compatibility.
// The canonical definitions now live in pkg/connector.
var (
	ErrNotImplemented       = connector.ErrNotImplemented
	ErrNotYetInstalled      = connector.ErrNotYetInstalled
	ErrInvalidClientRequest = connector.ErrInvalidClientRequest
	ErrUpstreamRatelimit    = connector.ErrUpstreamRatelimit
	ErrCurrencyNotSupported = connector.ErrCurrencyNotSupported
)
