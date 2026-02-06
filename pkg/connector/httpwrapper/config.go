package httpwrapper

import (
	"net/http"
	"time"

	"golang.org/x/oauth2/clientcredentials"
)

// Config holds the configuration for an HTTP client wrapper.
type Config struct {
	// HttpErrorCheckerFn is a custom function to check for HTTP errors.
	// If nil, a default checker is used that returns errors for 4xx and 5xx status codes.
	HttpErrorCheckerFn func(code int) error

	// Timeout is the request timeout. If zero, DefaultConnectorClientTimeout is used.
	Timeout time.Duration

	// Transport is the HTTP transport to use. If nil, http.DefaultTransport is used.
	Transport http.RoundTripper

	// OAuthConfig is optional OAuth2 client credentials configuration.
	OAuthConfig *clientcredentials.Config
}
