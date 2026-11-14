package metrics

import (
	"context"
	"net/http"
	"slices"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type MetricOpContextKey string

const MetricOperationContextKey MetricOpContextKey = "_metric_operation_context_key"

// unknownOperation is recorded when the caller did not set an operation with
// OperationContext.
const unknownOperation = "unknown"

func OperationContext(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, MetricOperationContextKey, operation)
}

type TransportOpts struct {
	Transport                http.RoundTripper
	CommonMetricAttributesFn func() []attribute.KeyValue
}

type Transport struct {
	connectorName          string
	parent                 http.RoundTripper
	commonMetricAttributes []attribute.KeyValue
}

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

func (r *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	res, err := r.parent.RoundTrip(req)
	registry := GetMetricsRegistry()

	// Labels must stay bounded: never record the request path, it carries
	// resource IDs and creates a new series for each one.
	operation := unknownOperation
	if name, ok := req.Context().Value(MetricOperationContextKey).(string); ok && name != "" {
		operation = name
	}

	attrs := slices.Clone(r.commonMetricAttributes)
	attrs = append(attrs,
		attribute.String("connector", r.connectorName),
		attribute.String("operation", operation),
	)

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
