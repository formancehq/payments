package client

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/formancehq/payments/genericclient"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type apiTransport struct {
	APIKey     string
	underlying http.RoundTripper
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", t.APIKey))

	return t.underlying.RoundTrip(req)
}

type Client struct {
	apiClient              *genericclient.APIClient
	commonMetricAttributes []attribute.KeyValue
}

func New(apiKey, baseURL string) *Client {
	httpClient := &http.Client{
		Transport: &apiTransport{
			APIKey:     apiKey,
			underlying: otelhttp.NewTransport(http.DefaultTransport),
		},
	}

	configuration := genericclient.NewConfiguration()
	configuration.HTTPClient = httpClient
	configuration.Servers[0].URL = baseURL

	genericClient := genericclient.NewAPIClient(configuration)

	return &Client{
		apiClient:              genericClient,
		commonMetricAttributes: CommonMetricsAttributes(),
	}
}

// recordMetrics is meant to be called in a defer
func (c *Client) recordMetrics(ctx context.Context, start time.Time, operation string) {
	registry := metrics.GetMetricsRegistry()

	attrs := c.commonMetricAttributes
	attrs = append(attrs, attribute.String("operation", operation))
	opts := metric.WithAttributes(attrs...)

	registry.ConnectorPSPCalls().Add(ctx, 1, opts)
	registry.ConnectorPSPCallLatencies().Record(ctx, time.Since(start).Milliseconds(), opts)
}

func CommonMetricsAttributes() []attribute.KeyValue {
	metricsAttributes := []attribute.KeyValue{
		attribute.String("connector", "generic"),
	}
	stack := os.Getenv("STACK")
	if stack != "" {
		metricsAttributes = append(metricsAttributes, attribute.String("stack", stack))
	}
	return metricsAttributes
}
