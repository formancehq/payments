package v3

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestOversizeRequestBody(t *testing.T) {
	t.Parallel()

	data := oversizeRequestBody()
	require.NotEmpty(t, data)
	require.Greater(t, len(data), 1000)
	
	for _, item := range data {
		require.Contains(t, item.Val, "Rindfleischetikettierungsüberwachungsaufgabenübertragungsgesetz")
	}
}

func TestGetQueryBuilder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("")
	_, span := tracer.Start(ctx, "test")
	defer span.End()

	t.Run("with body", func(t *testing.T) {
		t.Parallel()

		body := `{"$match": {"foo": "bar"}}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))

		qb, err := getQueryBuilder(span, req)
		require.NoError(t, err)
		require.NotNil(t, qb)
	})

	t.Run("with query parameter", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/test?query={\"$match\":{\"foo\":\"bar\"}}", nil)

		qb, err := getQueryBuilder(span, req)
		require.NoError(t, err)
		require.NotNil(t, qb)
	})

	t.Run("with empty body and no query parameter", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		qb, err := getQueryBuilder(span, req)
		require.NoError(t, err)
		require.NotNil(t, qb)
	})

	t.Run("with body read error", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPost, "/test", &errorReader{})

		_, err := getQueryBuilder(span, req)
		require.Error(t, err)
	})
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestGetPagination(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("")
	_, span := tracer.Start(ctx, "test")
	defer span.End()

	t.Run("with valid query", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/test?pageSize=10", nil)

		options, err := getPagination(span, req, struct{}{})
		require.NoError(t, err)
		require.NotNil(t, options)
		require.Equal(t, 10, options.PageSize)
	})

	t.Run("with invalid page size", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/test?pageSize=invalid", nil)

		_, err := getPagination(span, req, struct{}{})
		require.Error(t, err)
	})
}
