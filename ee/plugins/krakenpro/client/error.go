package client

import (
	"errors"
	"strings"
)

// ErrorResponse mirrors the Kraken wire envelope so the httpwrapper
// can decode 4xx/5xx bodies that carry the same JSON shape as success
// responses. ErrorResponse.Message picks the first error code.
type ErrorResponse struct {
	Errors []string `json:"error"`
}

// Message returns the first error code (or empty when none).
func (e ErrorResponse) Message() string {
	if len(e.Errors) == 0 {
		return ""
	}
	return e.Errors[0]
}

// APIError is the typed error carried by client methods when Kraken
// answers with a non-empty `error` array. Code is the first entry,
// All carries the full slice. Callers classify it via IsFatalAuthError
// (short-circuit retries) and IsRetriableError (rate-limit / nonce).
type APIError struct {
	Endpoint string
	Code     string
	All      []string
}

func (e *APIError) Error() string {
	return "krakenpro: " + e.Endpoint + ": " + strings.Join(e.All, "; ")
}

// IsAPIError tests whether err is (or wraps) an *APIError.
func IsAPIError(err error) bool {
	var a *APIError
	return errors.As(err, &a)
}

// fatalAuthCodes must not be retried — they mean a misconfigured
// connector (bad key/secret/permissions or a wrong endpoint), so retries
// only burn quota. Note "EAPI:Invalid nonce" is deliberately NOT here:
// with one API key shared across worker pods, out-of-order arrival makes
// the occasional invalid nonce a transient race, so it is retriable (see
// invalidNonceCode + IsRetriableError, and MAPPINGS for the per-worker
// key / nonce-window guidance).
var fatalAuthCodes = map[string]struct{}{
	"EAPI:Invalid key":           {},
	"EAPI:Invalid signature":     {},
	"EAPI:Bad request":           {},
	"EGeneral:Permission denied": {},
	"EGeneral:Unknown method":    {},
}

// IsFatalAuthError reports whether err is an APIError whose code
// is one of the documented fatal-auth codes.
func IsFatalAuthError(err error) bool {
	var a *APIError
	if !errors.As(err, &a) {
		return false
	}
	_, fatal := fatalAuthCodes[a.Code]
	return fatal
}

const invalidNonceCode = "EAPI:Invalid nonce"

// rateLimitCodes are the exact Kraken error codes that mean "back off".
// EService:Throttled carries a trailing timestamp ("EService:Throttled:
// 1700000000"), so it is matched by prefix in IsRateLimitError.
var rateLimitCodes = map[string]struct{}{
	"EAPI:Rate limit exceeded":   {},
	"EOrder:Rate limit exceeded": {},
	"EGeneral:Too many requests": {},
}

// IsRateLimitError reports whether err is an APIError signalling a
// rate limit / throttle. Kraken returns these in the error array
// (frequently on HTTP 200), so callers map them to the platform
// rate-limit path rather than relying on the HTTP status code.
func IsRateLimitError(err error) bool {
	var a *APIError
	if !errors.As(err, &a) {
		return false
	}
	if _, ok := rateLimitCodes[a.Code]; ok {
		return true
	}
	return strings.HasPrefix(a.Code, "EService:Throttled")
}

// IsRetriableError reports whether err is an APIError that should be
// retried with backoff: rate-limit/throttle, or a transient invalid
// nonce (a cross-pod race on a shared key). apiError routes them to
// backoff wrappers (rate-limit -> TooManyRequests, nonce -> TooEarly)
// so retries are spaced out — Kraken temp-locks a key after too many
// invalid-nonce errors.
func IsRetriableError(err error) bool {
	if IsRateLimitError(err) {
		return true
	}
	var a *APIError
	return errors.As(err, &a) && a.Code == invalidNonceCode
}
