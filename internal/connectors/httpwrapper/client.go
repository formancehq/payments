package httpwrapper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/internal/models"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/oauth2"
)

var (
	ErrStatusCodeUnexpected         = errors.New("unexpected status code")
	ErrStatusCodeClientError        = fmt.Errorf("%w: http client error", ErrStatusCodeUnexpected)
	ErrStatusCodeServerError        = fmt.Errorf("%w: http server error", ErrStatusCodeUnexpected)
	ErrStatusCodeTooManyRequests    = fmt.Errorf("%w: http too many requests error", ErrStatusCodeUnexpected)
	ErrStatusCodeRequestTimeout     = fmt.Errorf("%w: http request timeout", ErrStatusCodeUnexpected)
	ErrStatusCodeMisdirectedRequest = fmt.Errorf("%w: http misdirected request", ErrStatusCodeUnexpected)
	ErrStatusCodeLocked             = fmt.Errorf("%w: http locked", ErrStatusCodeUnexpected)
	ErrStatusCodeTooEarly           = fmt.Errorf("%w: http too early", ErrStatusCodeUnexpected)

	defaultHttpErrorCheckerFn = func(statusCode int) error {
		switch statusCode {
		case http.StatusTooManyRequests:
			return ErrStatusCodeTooManyRequests
		case http.StatusRequestTimeout:
			return ErrStatusCodeRequestTimeout
		case http.StatusMisdirectedRequest:
			return ErrStatusCodeMisdirectedRequest
		case http.StatusLocked:
			return ErrStatusCodeLocked
		case http.StatusTooEarly:
			return ErrStatusCodeTooEarly
		}

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
	httpClient *http.Client

	httpErrorCheckerFn func(statusCode int) error
}

func NewClient(config *Config) Client {
	if config.Timeout == 0 {
		config.Timeout = models.DefaultConnectorClientTimeout
	}
	if config.Transport == nil {
		config.Transport = http.DefaultTransport.(*http.Transport).Clone()
	}
	config.Transport = otelhttp.NewTransport(config.Transport)

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
		if errors.Is(err, context.DeadlineExceeded) {
			return 0, fmt.Errorf("%w: failed to make request: %w", ErrStatusCodeRequestTimeout, err)
		}
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			if urlErr.Timeout() {
				return 0, fmt.Errorf("%w: failed to make request: %w", ErrStatusCodeRequestTimeout, err)
			}
			if !errors.Is(err, context.Canceled) {
				return 0, fmt.Errorf("%w: failed to make request: %w", ErrStatusCodeClientError, err)
			}
		}
		return 0, fmt.Errorf("failed to make request: %w", err)
	}

	// Always drain and close the body, regardless of the return path below.
	// Draining lets the underlying connection be reused (keep-alive); closing
	// without draining (or not closing at all on early returns) leaks file
	// descriptors and connections in a long-running worker.
	// net/http guarantees resp.Body is non-nil on a successful Do, so there is
	// no nil check to perform here.
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			logging.FromContext(ctx).Errorf("failed to close response body: %v", closeErr)
		}
	}()

	reqErr := c.httpErrorCheckerFn(resp.StatusCode)
	// the caller doesn't care about the response body so we return early
	if (reqErr == nil && expectedBody == nil) || (reqErr != nil && errorBody == nil) {
		return resp.StatusCode, reqErr
	}

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
