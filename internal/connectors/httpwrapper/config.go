package httpwrapper

import (
	"net/http"
	"time"

	"golang.org/x/oauth2/clientcredentials"
)

type Config struct {
	HttpErrorCheckerFn func(code int) error

	Timeout     time.Duration
	Transport   http.RoundTripper
	OAuthConfig *clientcredentials.Config

	// DisableRateLimitHints opts OUT of rate-limit header parsing. By
	// default the client honours RFC 9110 Retry-After and
	// draft-ietf-httpapi-ratelimit-headers RateLimit on 429 / 5xx-with-
	// RateLimit-header responses, surfacing them as
	// *plugins.RateLimitedError so the engine's temporalPluginErrorCheck
	// feeds the hint into Temporal's NextRetryDelay (clamped to a
	// safety ceiling to defend against absurd values per RFC 9110 §8.4).
	// Set true only if a connector needs to bypass that path entirely.
	DisableRateLimitHints bool
}
