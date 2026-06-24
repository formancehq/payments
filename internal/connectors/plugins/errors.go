package plugins

import pkgplugins "github.com/formancehq/payments/pkg/domain/plugins"

var (
	ErrNotImplemented       = pkgplugins.ErrNotImplemented
	ErrNotYetInstalled      = pkgplugins.ErrNotYetInstalled
	ErrInvalidClientRequest = pkgplugins.ErrInvalidClientRequest
	ErrUpstreamRatelimit    = pkgplugins.ErrUpstreamRatelimit
	ErrUpstreamTimeout      = pkgplugins.ErrUpstreamTimeout
	ErrUpstreamRetryAfter   = pkgplugins.ErrUpstreamRetryAfter
	ErrCurrencyNotSupported = pkgplugins.ErrCurrencyNotSupported
)
