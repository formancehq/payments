package metrics

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

func NewHTTPClient(commonAttributesFn func() []attribute.KeyValue) *http.Client {
	opts := TransportOpts{
		CommonMetricAttributesFn: commonAttributesFn,
	}
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: NewTransport(opts),
	}
}
