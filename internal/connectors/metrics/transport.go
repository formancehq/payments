package metrics

import (
	"github.com/formancehq/payments/pkg/connector/metrics"
)

// MetricOpContextKey is the type for the metric operation context key.
type MetricOpContextKey = metrics.MetricOpContextKey

// MetricOperationContextKey is the context key used to store the operation name for metrics.
const MetricOperationContextKey = metrics.MetricOperationContextKey

// OperationContext returns a new context with the operation name set for metrics.
var OperationContext = metrics.OperationContext

// TransportOpts holds options for creating a metrics transport.
type TransportOpts = metrics.TransportOpts

// Transport is an http.RoundTripper that records metrics for each request.
type Transport = metrics.Transport

// NewTransport creates a new metrics transport for the given connector.
var NewTransport = metrics.NewTransport
