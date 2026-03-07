// Package metrics re-exports pkg/connector/metrics for internal use.
package metrics

import (
	"github.com/formancehq/payments/pkg/connector/metrics"
)

// MetricsRegistry provides access to connector metrics.
type MetricsRegistry = metrics.MetricsRegistry

// NoopMetricsRegistry is a no-op implementation of MetricsRegistry.
type NoopMetricsRegistry = metrics.NoopMetricsRegistry

// GetMetricsRegistry returns the global metrics registry.
var GetMetricsRegistry = metrics.GetMetricsRegistry

// RegisterMetricsRegistry creates and registers a new metrics registry.
var RegisterMetricsRegistry = metrics.RegisterMetricsRegistry

// NewNoOpMetricsRegistry creates a new no-op metrics registry.
var NewNoOpMetricsRegistry = metrics.NewNoOpMetricsRegistry
