package metrics

import (
    "context"
    "net/http"
    "time"
)

// Minimal metrics shim: no-op transport and client with same signatures used in connectors.

type MetricOpContextKey string

const MetricOperationContextKey MetricOpContextKey = "_metric_operation_context_key"

func OperationContext(ctx context.Context, operation string) context.Context {
    return context.WithValue(ctx, MetricOperationContextKey, operation)
}

type TransportOpts struct {
    Transport http.RoundTripper
}

type Transport struct {
    parent http.RoundTripper
}

func NewTransport(_ string, opts TransportOpts) http.RoundTripper {
    if opts.Transport == nil {
        opts.Transport = http.DefaultTransport
    }
    return &Transport{parent: opts.Transport}
}

func (r *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
    return r.parent.RoundTrip(req)
}

func NewHTTPClient(_ string, timeout time.Duration) *http.Client {
    return &http.Client{
        Timeout:   timeout,
        Transport: NewTransport("", TransportOpts{}),
    }
}

