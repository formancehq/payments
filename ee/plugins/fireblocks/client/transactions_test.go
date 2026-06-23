package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/formancehq/payments/pkg/domain/httpwrapper"
)

func TestListTransactionsPinsAfterOneOnFirstSync(t *testing.T) {
	t.Parallel()

	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	if _, err := c.ListTransactions(context.Background(), 0, 50); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// after=1 bypasses Fireblocks' "last 90 days" default.
	if !strings.Contains(capturedQuery, "after=1") {
		t.Fatalf("expected after=1 on initial sync, got %q", capturedQuery)
	}
	for _, want := range []string{"limit=50", "orderBy=createdAt", "sort=ASC"} {
		if !strings.Contains(capturedQuery, want) {
			t.Fatalf("expected query to contain %q, got %q", want, capturedQuery)
		}
	}
}

func TestListTransactionsForwardsStateCursor(t *testing.T) {
	t.Parallel()

	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	if _, err := c.ListTransactions(context.Background(), 1234567890, 50); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedQuery, "after=1234567890") {
		t.Fatalf("expected after=1234567890, got %q", capturedQuery)
	}
}

func TestListTransactionsRoundTripsTransactions(t *testing.T) {
	t.Parallel()

	body := []Transaction{
		{ID: "tx-1", AssetID: "BTC", Operation: "TRANSFER", Status: "COMPLETED", CreatedAt: 1700000000000},
		{ID: "tx-2", AssetID: "ETH", Operation: "TRANSFER", Status: "PENDING_SIGNATURE", CreatedAt: 1700000001000},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	txs, err := c.ListTransactions(context.Background(), 0, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(txs); got != 2 {
		t.Fatalf("expected 2 transactions, got %d", got)
	}
	if txs[0].ID != "tx-1" || txs[1].ID != "tx-2" {
		t.Fatalf("unexpected ids: %+v", txs)
	}
}

func TestListTransactionsWrapsServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"message":"boom","code":500}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	_, err := c.ListTransactions(context.Background(), 0, 50)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, httpwrapper.ErrStatusCodeServerError) {
		t.Fatalf("expected wrapped server-error, got %v", err)
	}
	if !strings.Contains(err.Error(), "failed to list transactions") {
		t.Fatalf("expected wrapping prefix, got %v", err)
	}
}
