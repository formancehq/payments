package metrics

import (
	"net/http"
	"time"
)

func NewHTTPClient(connectorName string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: NewTransport(connectorName, TransportOpts{}),
	}
}
