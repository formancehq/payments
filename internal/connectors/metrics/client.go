package metrics

import (
	"net/http"
	"time"
)

type ClientOptions struct {
	Timeout time.Duration
}

func NewHTTPClient(connectorName string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: NewTransport(connectorName, TransportOpts{}),
	}
}
