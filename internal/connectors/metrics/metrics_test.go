package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestGetMetricsRegistry(t *testing.T) {
	t.Parallel()

	registry = nil

	reg := GetMetricsRegistry()
	require.NotNil(t, reg)
	require.IsType(t, &NoopMetricsRegistry{}, reg)

	reg2 := GetMetricsRegistry()
	require.Equal(t, reg, reg2)
}

func TestRegisterMetricsRegistry(t *testing.T) {
	t.Parallel()

	registry = nil

	provider := noop.NewMeterProvider()
	reg, err := RegisterMetricsRegistry(provider)
	require.NoError(t, err)
	require.NotNil(t, reg)
	require.IsType(t, &metricsRegistry{}, reg)

	reg2 := GetMetricsRegistry()
	require.Equal(t, reg, reg2)

	counter := reg.ConnectorPSPCalls()
	require.NotNil(t, counter)

	histogram := reg.ConnectorPSPCallLatencies()
	require.NotNil(t, histogram)
}

func TestNoOpMetricsRegistry(t *testing.T) {
	t.Parallel()

	reg := NewNoOpMetricsRegistry()
	require.NotNil(t, reg)

	counter := reg.ConnectorPSPCalls()
	require.NotNil(t, counter)

	histogram := reg.ConnectorPSPCallLatencies()
	require.NotNil(t, histogram)
}
