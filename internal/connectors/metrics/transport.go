package metrics

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const MetricOperationContextKey string = "_metric_operation_context_key"

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
