// Package httpwrapper provides a convenience HTTP client wrapper for connector implementations.
package httpwrapper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connector"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/oauth2"
)

var (
	// ErrStatusCodeUnexpected is returned when an unexpected HTTP status code is received.
	ErrStatusCodeUnexpected = errors.New("unexpected status code")
	// ErrStatusCodeClientError is returned for 4xx HTTP status codes.
	ErrStatusCodeClientError = fmt.Errorf("%w: http client error", ErrStatusCodeUnexpected)
	// ErrStatusCodeServerError is returned for 5xx HTTP status codes.
	ErrStatusCodeServerError = fmt.Errorf("%w: http server error", ErrStatusCodeUnexpected)
	// ErrStatusCodeTooManyRequests is returned for HTTP 429 status code.
	ErrStatusCodeTooManyRequests = fmt.Errorf("%w: http too many requests error", ErrStatusCodeUnexpected)

	defaultHttpErrorCheckerFn = func(statusCode int) error {
		if statusCode == http.StatusTooManyRequests {
			return ErrStatusCodeTooManyRequests
		}

		if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
			return ErrStatusCodeClientError
		} else if statusCode >= http.StatusInternalServerError {
			return ErrStatusCodeServerError
		}
		return nil
	}
)

// Client is a convenience wrapper that encapsulates common code related to interacting with HTTP endpoints.
type Client interface {
	// Do performs an HTTP request while handling errors and unmarshaling success and error responses into the provided interfaces.
	// expectedBody and errorBody should be pointers to structs.
	Do(ctx context.Context, req *http.Request, expectedBody, errorBody any) (statusCode int, err error)
}

type client struct {
	httpClient *http.Client

	httpErrorCheckerFn func(statusCode int) error
}

// NewClient creates a new HTTP client wrapper with the given configuration.
func NewClient(config *Config) Client {
	if config.Timeout == 0 {
		config.Timeout = connector.DefaultConnectorClientTimeout
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

	return &client{
		httpErrorCheckerFn: config.HttpErrorCheckerFn,
		httpClient:         httpClient,
	}
}

func (c *client) Do(ctx context.Context, req *http.Request, expectedBody, errorBody any) (int, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}

	reqErr := c.httpErrorCheckerFn(resp.StatusCode)
	// the caller doesn't care about the response body so we return early
	if resp.Body == nil || (reqErr == nil && expectedBody == nil) || (reqErr != nil && errorBody == nil) {
		return resp.StatusCode, reqErr
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			logging.FromContext(ctx).Errorf("failed to close response body: %w", err)
		}
	}()

	// TODO: reading everything into memory might not be optimal if we expect long responses
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	if reqErr != nil {
		if err = json.Unmarshal(rawBody, errorBody); err != nil {
			return resp.StatusCode, fmt.Errorf("failed to unmarshal error response (%w) with status %d: %w", err, resp.StatusCode, reqErr)
		}
		return resp.StatusCode, reqErr
	}

	// TODO: assuming json bodies for now, but may need to handle other body types
	if err = json.Unmarshal(rawBody, expectedBody); err != nil {
		return resp.StatusCode, fmt.Errorf("failed to unmarshal response with status %d: %w", resp.StatusCode, err)
	}
	return resp.StatusCode, nil
}
