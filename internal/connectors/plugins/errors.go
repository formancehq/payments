package plugins

import (
	"errors"
)

var (
	ErrNotImplemented       = errors.New("not implemented")
	ErrNotYetInstalled      = errors.New("not yet installed")
	ErrInvalidClientRequest = errors.New("invalid client request")
	ErrUpstreamRatelimit    = errors.New("rate limited by upstream server")
	ErrUpstreamTimeout      = errors.New("upstream timeout")
	ErrUpstreamRetryAfter   = errors.New("upstream asked to retry later")
	ErrCurrencyNotSupported = errors.New("currency not supported")
)
