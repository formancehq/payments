package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/formancehq/payments/pkg/domain/httpwrapper"
)

func TestListBlockchainsPaginatesUsingCursor(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	var capturedQueries []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQueries = append(capturedQueries, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		switch calls.Add(1) {
		case 1:
			_ = json.NewEncoder(w).Encode(BlockchainsResponse{
				Data: []Blockchain{
					{ID: "chain-1", LegacyID: "ETH", Onchain: &BlockchainOnchain{Test: false}},
					{ID: "chain-2", LegacyID: "ETH_TEST5", Onchain: &BlockchainOnchain{Test: true}},
				},
				Next: "cursor-2",
			})
		default:
			_ = json.NewEncoder(w).Encode(BlockchainsResponse{
				Data: []Blockchain{{ID: "chain-3", LegacyID: "BTC"}},
				Next: "",
			})
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	chains, err := c.ListBlockchains(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(chains); got != 3 {
		t.Fatalf("expected 3 blockchains across 2 pages, got %d", got)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 HTTP calls, got %d", calls.Load())
	}
	if !strings.Contains(capturedQueries[0], "pageSize=500") || strings.Contains(capturedQueries[0], "pageCursor=") {
		t.Fatalf("first request should have pageSize and no cursor, got %q", capturedQueries[0])
	}
	if !strings.Contains(capturedQueries[1], "pageCursor=cursor-2") {
		t.Fatalf("second request should carry pageCursor=cursor-2, got %q", capturedQueries[1])
	}
}

func TestListBlockchainsWrapsServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"boom","code":500}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	_, err := c.ListBlockchains(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, httpwrapper.ErrStatusCodeServerError) {
		t.Fatalf("expected wrapped server-error, got %v", err)
	}
	if !strings.Contains(err.Error(), "failed to list blockchains") {
		t.Fatalf("expected wrapping prefix, got %v", err)
	}
}
