package metrics

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

type recordingCounter struct {
	noop.Int64Counter
	attrs []attribute.Set
}

func (c *recordingCounter) Add(_ context.Context, _ int64, opts ...metric.AddOption) {
	c.attrs = append(c.attrs, metric.NewAddConfig(opts).Attributes())
}

type recordingHistogram struct {
	noop.Int64Histogram
	attrs []attribute.Set
}

func (h *recordingHistogram) Record(_ context.Context, _ int64, opts ...metric.RecordOption) {
	h.attrs = append(h.attrs, metric.NewRecordConfig(opts).Attributes())
}

type recordingRegistry struct {
	calls     *recordingCounter
	latencies *recordingHistogram
}

func (r *recordingRegistry) ConnectorPSPCalls() metric.Int64Counter           { return r.calls }
func (r *recordingRegistry) ConnectorPSPCallLatencies() metric.Int64Histogram { return r.latencies }

type stubRoundTripper struct {
	err error
}

func (s stubRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

func useRecordingRegistry(t *testing.T) *recordingRegistry {
	t.Helper()
	rec := &recordingRegistry{calls: &recordingCounter{}, latencies: &recordingHistogram{}}
	registryMu.Lock()
	previous := registry
	registry = rec
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		registry = previous
		registryMu.Unlock()
	})
	return rec
}

func TestTransportLabelsAreBounded(t *testing.T) {
	tests := []struct {
		name          string
		ctx           context.Context
		path          string
		parentErr     error
		wantOperation string
		wantStatus    int
	}{
		{
			name:          "operation set",
			ctx:           OperationContext(context.Background(), "get_wallet"),
			path:          "/v1/wallets/570270d8-0b7d-4c4e-9d3a-1f0e8c6f9a11",
			wantOperation: "get_wallet",
			wantStatus:    http.StatusOK,
		},
		{
			name:          "operation missing",
			ctx:           context.Background(),
			path:          "/v1/wallets/wlt_m_01JE0000000000000000000000",
			wantOperation: unknownOperation,
			wantStatus:    http.StatusOK,
		},
		{
			name:          "operation empty",
			ctx:           OperationContext(context.Background(), ""),
			path:          "/v1/wallets/123",
			wantOperation: unknownOperation,
			wantStatus:    http.StatusOK,
		},
		{
			name:          "operation of wrong type",
			ctx:           context.WithValue(context.Background(), MetricOperationContextKey, 42),
			path:          "/v1/wallets/123",
			wantOperation: unknownOperation,
			wantStatus:    http.StatusOK,
		},
		{
			name:          "transport error",
			ctx:           OperationContext(context.Background(), "get_wallet"),
			path:          "/v1/wallets/123",
			parentErr:     errors.New("connection refused"),
			wantOperation: "get_wallet",
			wantStatus:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := useRecordingRegistry(t)
			transport := NewTransport("test", TransportOpts{
				Transport: stubRoundTripper{err: tt.parentErr},
				CommonMetricAttributesFn: func() []attribute.KeyValue {
					return []attribute.KeyValue{attribute.String("common", "value")}
				},
			})

			req, err := http.NewRequestWithContext(tt.ctx, http.MethodGet, "https://psp.example"+tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			_, err = transport.RoundTrip(req)
			if !errors.Is(err, tt.parentErr) {
				t.Fatalf("unexpected error: %v", err)
			}

			want := attribute.NewSet(
				attribute.String("common", "value"),
				attribute.String("connector", "test"),
				attribute.String("operation", tt.wantOperation),
				attribute.Int("status", tt.wantStatus),
			)
			for name, recorded := range map[string][]attribute.Set{
				"calls":     rec.calls.attrs,
				"latencies": rec.latencies.attrs,
			} {
				if len(recorded) != 1 {
					t.Fatalf("%s: expected 1 recording, got %d", name, len(recorded))
				}
				if !recorded[0].Equals(&want) {
					t.Fatalf("%s: unexpected attributes:\n got: %v\nwant: %v", name, recorded[0].ToSlice(), want.ToSlice())
				}
			}
		})
	}
}

// TestTransportDoesNotMutateCommonAttributes guards against appending to the
// shared common attribute slice: with spare capacity, concurrent requests would
// overwrite each other's labels.
func TestTransportDoesNotMutateCommonAttributes(t *testing.T) {
	useRecordingRegistry(t)

	common := make([]attribute.KeyValue, 1, 8)
	common[0] = attribute.String("common", "value")
	backing := common[:cap(common)]

	transport := NewTransport("test", TransportOpts{
		Transport:                stubRoundTripper{},
		CommonMetricAttributesFn: func() []attribute.KeyValue { return common },
	})

	req, err := http.NewRequestWithContext(OperationContext(context.Background(), "get_wallet"), http.MethodGet, "https://psp.example/v1/wallets", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatal(err)
	}

	for i, kv := range backing[1:] {
		if kv.Valid() {
			t.Fatalf("common attribute backing array was written at index %d: %v", i+1, kv)
		}
	}
}
