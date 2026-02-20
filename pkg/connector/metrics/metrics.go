// Package metrics provides OpenTelemetry metrics for connector PSP calls.
package metrics

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

var registry MetricsRegistry

// GetMetricsRegistry returns the global metrics registry.
// If no registry has been registered, it returns a no-op implementation.
func GetMetricsRegistry() MetricsRegistry {
	if registry == nil {
		registry = NewNoOpMetricsRegistry()
	}

	return registry
}

// MetricsRegistry provides access to connector metrics.
type MetricsRegistry interface {
	// ConnectorPSPCalls returns the counter for PSP API calls.
	ConnectorPSPCalls() metric.Int64Counter
	// ConnectorPSPCallLatencies returns the histogram for PSP API call latencies.
	ConnectorPSPCallLatencies() metric.Int64Histogram
}

type metricsRegistry struct {
	connectorPSPCalls         metric.Int64Counter
	connectorPSPCallLatencies metric.Int64Histogram
}

// RegisterMetricsRegistry creates and registers a new metrics registry with the given meter provider.
func RegisterMetricsRegistry(meterProvider metric.MeterProvider) (MetricsRegistry, error) {
	meter := meterProvider.Meter("payments")

	connectorPSPCalls, err := meter.Int64Counter(
		"payments_connectors_psp_calls",
		metric.WithUnit("1"),
		metric.WithDescription("payments connectors psp calls"),
	)
	if err != nil {
		return nil, err
	}

	connectorPSPCallLatencies, err := meter.Int64Histogram(
		"payments_connectors_psp_calls_latencies",
		metric.WithUnit("ms"),
		metric.WithDescription("payments connectors psp calls latencies"),
	)
	if err != nil {
		return nil, err
	}

	registry = &metricsRegistry{
		connectorPSPCalls:         connectorPSPCalls,
		connectorPSPCallLatencies: connectorPSPCallLatencies,
	}

	return registry, nil
}

func (m *metricsRegistry) ConnectorPSPCalls() metric.Int64Counter {
	return m.connectorPSPCalls
}

func (m *metricsRegistry) ConnectorPSPCallLatencies() metric.Int64Histogram {
	return m.connectorPSPCallLatencies
}

// NoopMetricsRegistry is a no-op implementation of MetricsRegistry.
type NoopMetricsRegistry struct{}

// NewNoOpMetricsRegistry creates a new no-op metrics registry.
func NewNoOpMetricsRegistry() *NoopMetricsRegistry {
	return &NoopMetricsRegistry{}
}

func (m *NoopMetricsRegistry) ConnectorPSPCalls() metric.Int64Counter {
	counter, _ := noop.NewMeterProvider().Meter("payments").Int64Counter("payments_connectors_psp_calls")
	return counter
}

func (m *NoopMetricsRegistry) ConnectorPSPCallLatencies() metric.Int64Histogram {
	histogram, _ := noop.NewMeterProvider().Meter("payments").Int64Histogram("payments_connectors_psp_calls_latencies")
	return histogram
}

var (
	_ MetricsRegistry = (*metricsRegistry)(nil)
	_ MetricsRegistry = (*NoopMetricsRegistry)(nil)
)
