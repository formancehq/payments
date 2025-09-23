package http

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "time"

    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
    ErrStatusCodeUnexpected      = errors.New("unexpected status code")
    ErrStatusCodeClientError     = fmt.Errorf("%w: http client error", ErrStatusCodeUnexpected)
    ErrStatusCodeServerError     = fmt.Errorf("%w: http server error", ErrStatusCodeUnexpected)
    ErrStatusCodeTooManyRequests = fmt.Errorf("%w: http too many requests error", ErrStatusCodeUnexpected)
)

type Config struct {
    HttpErrorCheckerFn func(code int) error
    Timeout            time.Duration
    Transport          http.RoundTripper
}

type Client interface {
    Do(ctx context.Context, req *http.Request, expectedBody, errorBody any) (statusCode int, err error)
}

type client struct {
    httpClient         *http.Client
    httpErrorCheckerFn func(statusCode int) error
}

func NewClient(config *Config) Client {
    if config.Timeout == 0 {
        config.Timeout = 10 * time.Second
    }
    if config.Transport != nil {
        config.Transport = otelhttp.NewTransport(config.Transport)
    } else {
        config.Transport = http.DefaultTransport.(*http.Transport).Clone()
    }
    if config.HttpErrorCheckerFn == nil {
        config.HttpErrorCheckerFn = func(code int) error {
            if code == http.StatusTooManyRequests {
                return ErrStatusCodeTooManyRequests
            }
            if code >= http.StatusBadRequest && code < http.StatusInternalServerError {
                return ErrStatusCodeClientError
            } else if code >= http.StatusInternalServerError {
                return ErrStatusCodeServerError
            }
            return nil
        }
    }
    httpClient := &http.Client{Timeout: config.Timeout, Transport: config.Transport}
    return &client{httpClient: httpClient, httpErrorCheckerFn: config.HttpErrorCheckerFn}
}

func (c *client) Do(ctx context.Context, req *http.Request, expectedBody, errorBody any) (int, error) {
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return 0, fmt.Errorf("failed to make request: %w", err)
    }
    reqErr := c.httpErrorCheckerFn(resp.StatusCode)
    if resp.Body == nil || (reqErr == nil && expectedBody == nil) || (reqErr != nil && errorBody == nil) {
        return resp.StatusCode, reqErr
    }
    defer resp.Body.Close()
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
    if err = json.Unmarshal(rawBody, expectedBody); err != nil {
        return resp.StatusCode, fmt.Errorf("failed to unmarshal response with status %d: %w", resp.StatusCode, err)
    }
    return resp.StatusCode, nil
}

