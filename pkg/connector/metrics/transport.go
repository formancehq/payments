package metrics

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricOpContextKey is the type for the metric operation context key.
type MetricOpContextKey string

// MetricOperationContextKey is the context key used to store the operation name for metrics.
const MetricOperationContextKey MetricOpContextKey = "_metric_operation_context_key"

// OperationContext returns a new context with the operation name set for metrics.
func OperationContext(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, MetricOperationContextKey, operation)
}

// TransportOpts holds options for creating a metrics transport.
type TransportOpts struct {
	// Transport is the underlying HTTP transport. If nil, http.DefaultTransport is used.
	Transport http.RoundTripper
	// CommonMetricAttributesFn returns common attributes to add to all metrics.
	CommonMetricAttributesFn func() []attribute.KeyValue
}

// Transport is an http.RoundTripper that records metrics for each request.
type Transport struct {
	connectorName          string
	parent                 http.RoundTripper
	commonMetricAttributes []attribute.KeyValue
}

// NewTransport creates a new metrics transport for the given connector.
func NewTransport(connectorName string, opts TransportOpts) http.RoundTripper {
	if opts.Transport == nil {
		opts.Transport = http.DefaultTransport
	}

	if opts.CommonMetricAttributesFn == nil {
		opts.CommonMetricAttributesFn = func() []attribute.KeyValue { return []attribute.KeyValue{} }
	}

	return &Transport{
		connectorName:          connectorName,
		parent:                 opts.Transport,
		commonMetricAttributes: opts.CommonMetricAttributesFn(),
	}
}

// RoundTrip implements http.RoundTripper and records metrics for the request.
func (r *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	res, err := r.parent.RoundTrip(req)
	registry := GetMetricsRegistry()

	attrs := r.commonMetricAttributes
	attrs = append(attrs, attribute.String("connector", r.connectorName))
	attrs = append(attrs, attribute.String("endpoint", req.URL.Path))
	if val := req.Context().Value(MetricOperationContextKey); val != nil {
		if name, ok := val.(string); ok {
			attrs = append(attrs, attribute.String("operation", name))
		}
	}

	if res != nil {
		attrs = append(attrs, attribute.Int("status", res.StatusCode))
	} else {
		// if request could not be executed
		attrs = append(attrs, attribute.Int("status", 0))
	}
	opts := metric.WithAttributes(attrs...)

	registry.ConnectorPSPCalls().Add(req.Context(), 1, opts)
	registry.ConnectorPSPCallLatencies().Record(req.Context(), time.Since(start).Milliseconds(), opts)
	return res, err
}
