package httpwrapper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
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

	httpErrorCheckerFn    func(statusCode int) error
	disableRateLimitHints bool
}

func NewClient(config *Config) Client {
	if config.Timeout == 0 {
		config.Timeout = models.DefaultConnectorClientTimeout
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
		httpErrorCheckerFn:    config.HttpErrorCheckerFn,
		httpClient:            httpClient,
		disableRateLimitHints: config.DisableRateLimitHints,
	}
}

func (c *client) Do(ctx context.Context, req *http.Request, expectedBody, errorBody any) (int, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}

	reqErr := c.httpErrorCheckerFn(resp.StatusCode)
	rateLimited, retryAfter := false, time.Duration(0)
	if !c.disableRateLimitHints {
		rateLimited, retryAfter = classifyRateLimitResponse(resp)
	}

	// the caller doesn't care about the response body so we return early
	if resp.Body == nil || (reqErr == nil && expectedBody == nil) || (reqErr != nil && errorBody == nil) {
		return resp.StatusCode, c.maybeWrapRateLimit(reqErr, rateLimited, retryAfter)
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
		// Empty error bodies are common on bare 429s and some 5xx
		// responses; skip the unmarshal in that case so we don't lose
		// the rate-limit classification to a "unexpected end of JSON
		// input" failure.
		if len(rawBody) > 0 {
			if err = json.Unmarshal(rawBody, errorBody); err != nil {
				return resp.StatusCode, fmt.Errorf("failed to unmarshal error response (%w) with status %d: %w", err, resp.StatusCode, reqErr)
			}
		}
		return resp.StatusCode, c.maybeWrapRateLimit(reqErr, rateLimited, retryAfter)
	}

	// TODO: assuming json bodies for now, but may need to handle other body types
	if err = json.Unmarshal(rawBody, expectedBody); err != nil {
		return resp.StatusCode, fmt.Errorf("failed to unmarshal response with status %d: %w", resp.StatusCode, err)
	}
	return resp.StatusCode, nil
}

// maybeWrapRateLimit upgrades a status-driven error into
// *plugins.RateLimitedError when the response was classified as
// rate-limited. The wrapping keeps errors.Is(err, ErrUpstreamRatelimit)
// AND errors.Is(err, ErrStatusCode*) both satisfied, so existing
// connector code that branches on either sentinel keeps working.
func (c *client) maybeWrapRateLimit(reqErr error, rateLimited bool, retryAfter time.Duration) error {
	if !rateLimited || reqErr == nil {
		return reqErr
	}
	return plugins.NewRateLimitedError(retryAfter, reqErr)
}
