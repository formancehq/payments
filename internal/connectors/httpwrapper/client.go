package httpwrapper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/hashicorp/go-hclog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/oauth2"
)

const MetricOperationContextKey = "_metric_operation_context_key"

var (
	ErrStatusCodeUnexpected  = errors.New("unexpected status code")
	ErrStatusCodeClientError = fmt.Errorf("%w: http client error", ErrStatusCodeUnexpected)
	ErrStatusCodeServerError = fmt.Errorf("%w: http server error", ErrStatusCodeUnexpected)

	defaultHttpErrorCheckerFn = func(statusCode int) error {
		if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
			return ErrStatusCodeClientError
		} else if statusCode >= http.StatusInternalServerError {
			return ErrStatusCodeServerError
		}
		return nil
	}
)

// Client is a convenience wrapper that encapsulates common code related to interacting with HTTP endpoints
type Client interface {
	// Do performs an HTTP request while handling errors and unmarshaling success and error responses into the provided interfaces
	// expectedBody and errorBody should be pointers to structs
	Do(ctx context.Context, req *http.Request, expectedBody, errorBody any) (statusCode int, err error)
}

type client struct {
	httpClient              *http.Client
	commonMetricsAttributes []attribute.KeyValue

	httpErrorCheckerFn func(statusCode int) error
}

func NewClient(config *Config) (Client, error) {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	if config.Transport != nil {
		config.Transport = otelhttp.NewTransport(config.Transport)
	} else {
		config.Transport = http.DefaultTransport.(*http.Transport).Clone()
	}

	httpClient := &http.Client{
		Timeout:   config.Timeout,
		Transport: config.Transport,
	}
	if config.OAuthConfig != nil {
		// pass a pre-configured http client to oauth lib via the context
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
		httpClient = config.OAuthConfig.Client(ctx)
	}

	if config.HttpErrorCheckerFn == nil {
		config.HttpErrorCheckerFn = defaultHttpErrorCheckerFn
	}

	metricsAttributes := make([]attribute.KeyValue, 0)
	for i := range config.CommonMetricsAttributes {
		metricsAttributes = append(metricsAttributes, config.CommonMetricsAttributes[i])
	}

	return &client{
		httpErrorCheckerFn:      config.HttpErrorCheckerFn,
		httpClient:              httpClient,
		commonMetricsAttributes: metricsAttributes,
	}, nil
}

func (c *client) Do(ctx context.Context, req *http.Request, expectedBody, errorBody any) (int, error) {
	start := time.Now()
	attrs := c.metricsAttributes(ctx, req)
	defer func() {
		registry := metrics.GetMetricsRegistry()
		opts := metric.WithAttributes(attrs...)
		registry.ConnectorPSPCalls().Add(ctx, 1, opts)
		registry.ConnectorPSPCallLatencies().Record(ctx, time.Since(start).Milliseconds(), opts)
	}()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}
	attrs = append(attrs, attribute.Int("status", resp.StatusCode))

	reqErr := c.httpErrorCheckerFn(resp.StatusCode)
	// the caller doesn't care about the response body so we return early
	if resp.Body == nil || (reqErr == nil && expectedBody == nil) || (reqErr != nil && errorBody == nil) {
		return resp.StatusCode, reqErr
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			hclog.Default().Error("failed to close response body", "error", err)
		}
	}()

	// TODO: reading everything into memory might not be optimal if we expect long responses
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	if reqErr != nil {
		if err = json.Unmarshal(rawBody, errorBody); err != nil {
			return resp.StatusCode, fmt.Errorf("failed to unmarshal error response with status %d: %w", resp.StatusCode, err)
		}
		return resp.StatusCode, reqErr
	}

	// TODO: assuming json bodies for now, but may need to handle other body types
	if err = json.Unmarshal(rawBody, expectedBody); err != nil {
		return resp.StatusCode, fmt.Errorf("failed to unmarshal response with status %d: %w", resp.StatusCode, err)
	}
	return resp.StatusCode, nil
}

func (c *client) metricsAttributes(ctx context.Context, req *http.Request) []attribute.KeyValue {
	attrs := c.commonMetricsAttributes
	attrs = append(attrs, attribute.String("endpoint", req.URL.Path))

	val := ctx.Value(MetricOperationContextKey)
	if val != nil {
		if name, ok := val.(string); ok {
			attrs = append(attrs, attribute.String("name", name))
		}
	}
	return attrs
}
