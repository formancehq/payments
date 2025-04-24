package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestOperationContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	const operation = "test-operation"

	newCtx := OperationContext(ctx, operation)
	require.NotEqual(t, ctx, newCtx)

	value := newCtx.Value(MetricOperationContextKey)
	require.Equal(t, operation, value)
}

type mockTransport struct {
	response *http.Response
	err      error
}

func (m *mockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return m.response, m.err
}

func TestNewTransport(t *testing.T) {
	t.Parallel()

	transport := NewTransport("test-connector", TransportOpts{})
	require.NotNil(t, transport)
	require.IsType(t, &Transport{}, transport)
	tr := transport.(*Transport)
	require.Equal(t, "test-connector", tr.connectorName)
	require.Equal(t, http.DefaultTransport, tr.parent)
	require.NotNil(t, tr.commonMetricAttributes)
	require.Len(t, tr.commonMetricAttributes, 0)

	mockTr := &mockTransport{
		response: &http.Response{StatusCode: http.StatusOK},
	}
	customAttrs := []attribute.KeyValue{attribute.String("custom", "attr")}
	transport = NewTransport("test-connector", TransportOpts{
		Transport: mockTr,
		CommonMetricAttributesFn: func() []attribute.KeyValue {
			return customAttrs
		},
	})

	require.NotNil(t, transport)
	tr = transport.(*Transport)
	require.Equal(t, mockTr, tr.parent)
	require.Equal(t, customAttrs, tr.commonMetricAttributes)
}

func TestTransportRoundTrip(t *testing.T) {
	t.Parallel()

	registry = nil

	mockResp := &http.Response{StatusCode: http.StatusOK}
	mockTr := &mockTransport{
		response: mockResp,
	}

	transport := NewTransport("test-connector", TransportOpts{
		Transport: mockTr,
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/path", nil)
	ctx := OperationContext(context.Background(), "test-operation")
	req = req.WithContext(ctx)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, mockResp, resp)

	mockTr.response = nil
	mockTr.err = http.ErrHandlerTimeout
	resp, err = transport.RoundTrip(req)
	require.Error(t, err)
	require.Equal(t, http.ErrHandlerTimeout, err)
	require.Nil(t, resp)
}

func TestNewHTTPClient(t *testing.T) {
	t.Parallel()

	timeout := 30 * time.Second
	client := NewHTTPClient("test-connector", timeout)
	require.NotNil(t, client)
	require.Equal(t, timeout, client.Timeout)
	require.IsType(t, &Transport{}, client.Transport)

	tr := client.Transport.(*Transport)
	require.Equal(t, "test-connector", tr.connectorName)
}
