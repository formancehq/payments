package plugins

import (
	"errors"
	"fmt"
	"time"
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

// RateLimitedError carries an upstream wait hint alongside the
// ErrUpstreamRatelimit sentinel. Plugins parse it from RFC 9110
// Retry-After and/or draft-ietf-httpapi-ratelimit-headers RateLimit
// headers; the engine's temporalPluginErrorCheck feeds RetryAfter into
// Temporal's per-error NextRetryDelay. Wraps ErrUpstreamRatelimit so
// existing errors.Is(err, ErrUpstreamRatelimit) call sites stay green.
type RateLimitedError struct {
	// RetryAfter: zero means no hint (engine falls back to its static delay).
	RetryAfter time.Duration
	// Cause is the underlying transport error, preserved for logging.
	Cause error
}

func (e *RateLimitedError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s", ErrUpstreamRatelimit.Error(), e.Cause.Error())
	}
	return ErrUpstreamRatelimit.Error()
}

func (e *RateLimitedError) Unwrap() []error {
	if e.Cause != nil {
		return []error{ErrUpstreamRatelimit, e.Cause}
	}
	return []error{ErrUpstreamRatelimit}
}

// NewRateLimitedError treats negative retryAfter as "no hint" so callers
// can pass a parsed-but-absent header value directly.
func NewRateLimitedError(retryAfter time.Duration, cause error) *RateLimitedError {
	if retryAfter < 0 {
		retryAfter = 0
	}
	return &RateLimitedError{RetryAfter: retryAfter, Cause: cause}
}
