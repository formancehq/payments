package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetBalancesForSymbolFiltersCaseInsensitive(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/portfolios/portfolio-123/balances" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"balances": [
				{"symbol":"BTC","amount":"1.5","holds":"0.5","withdrawable_amount":"1.0","fiat_amount":"75000.00"},
				{"symbol":"USD","amount":"1000.50","holds":"100.50","withdrawable_amount":"900.00","fiat_amount":"1000.50"}
			],
			"pagination": {"next_cursor":"","sort_direction":"ASC","has_next":false}
		}`))
	}))
	defer server.Close()

	c := NewWithBaseURL("coinbase", "api-key", "dGVzdA==", "passphrase", "portfolio-123", server.URL)

	response, err := c.GetBalancesForSymbol(context.Background(), "btc", "", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Balances) != 1 {
		t.Fatalf("expected 1 balance, got %d", len(response.Balances))
	}

	if response.Balances[0].Symbol != "BTC" {
		t.Fatalf("expected BTC symbol, got %s", response.Balances[0].Symbol)
	}
}

func TestGetBalancesForSymbolRequiresSymbol(t *testing.T) {
	t.Parallel()

	c := NewWithBaseURL("coinbase", "api-key", "dGVzdA==", "passphrase", "portfolio-123", "http://localhost")

	_, err := c.GetBalancesForSymbol(context.Background(), "   ", "", 100)
	if err == nil {
		t.Fatalf("expected an error when symbol is missing")
	}
}

func TestPortfolioEndpointsEncodeCursor(t *testing.T) {
	t.Parallel()

	const (
		portfolioID = "portfolio-123"
		pageSize    = 50
	)
	cursor := "abc+def&x=1 value"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "50" {
			t.Fatalf("unexpected limit query param: %q", got)
		}
		if got := r.URL.Query().Get("sort_direction"); got != "ASC" {
			t.Fatalf("unexpected sort_direction query param: %q", got)
		}
		if got := r.URL.Query().Get("cursor"); got != cursor {
			t.Fatalf("unexpected cursor query param: %q", got)
		}
		if got := r.URL.Query().Get("x"); got != "" {
			t.Fatalf("cursor leaked into separate query param x=%q", got)
		}
		if strings.Contains(r.URL.RawQuery, "cursor="+cursor) {
			t.Fatalf("cursor should be URL-encoded, raw query was %q", r.URL.RawQuery)
		}

		switch r.URL.Path {
		case "/v1/portfolios/" + portfolioID + "/wallets":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"wallets":[],"pagination":{"next_cursor":"","sort_direction":"ASC","has_next":false}}`))
		case "/v1/portfolios/" + portfolioID + "/balances":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"balances":[],"pagination":{"next_cursor":"","sort_direction":"ASC","has_next":false}}`))
		case "/v1/portfolios/" + portfolioID + "/transactions":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"transactions":[],"pagination":{"next_cursor":"","sort_direction":"ASC","has_next":false}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewWithBaseURL("coinbase", "api-key", "dGVzdA==", "passphrase", portfolioID, server.URL)

	if _, err := c.GetWallets(context.Background(), cursor, pageSize); err != nil {
		t.Fatalf("GetWallets failed: %v", err)
	}
	if _, err := c.GetBalances(context.Background(), cursor, pageSize); err != nil {
		t.Fatalf("GetBalances failed: %v", err)
	}
	if _, err := c.GetTransactions(context.Background(), cursor, pageSize); err != nil {
		t.Fatalf("GetTransactions failed: %v", err)
	}
}
