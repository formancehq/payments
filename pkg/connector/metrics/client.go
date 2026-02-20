package metrics

import (
	"net/http"
	"time"
)

// NewHTTPClient creates a new HTTP client with metrics instrumentation.
func NewHTTPClient(connectorName string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: NewTransport(connectorName, TransportOpts{}),
	}
}
