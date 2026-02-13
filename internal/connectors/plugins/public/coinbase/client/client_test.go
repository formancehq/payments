package client

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSignRequestBase64DecodesSecret(t *testing.T) {
	t.Parallel()

	// Known base64-encoded secret
	secret := base64.StdEncoding.EncodeToString([]byte("my-secret-key"))

	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"wallets":[],"pagination":{"next_cursor":"","sort_direction":"ASC","has_next":false}}`))
	}))
	defer server.Close()

	c := NewWithBaseURL("coinbase", "api-key", secret, "passphrase", "portfolio-123", server.URL)

	_, err := c.GetWallets(context.Background(), "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sig := capturedHeaders.Get("X-CB-ACCESS-SIGNATURE")
	if sig == "" {
		t.Fatal("expected X-CB-ACCESS-SIGNATURE header to be set")
	}

	timestamp := capturedHeaders.Get("X-CB-ACCESS-TIMESTAMP")
	if timestamp == "" {
		t.Fatal("expected X-CB-ACCESS-TIMESTAMP header to be set")
	}

	// Verify the signature was computed with the decoded secret bytes
	message := timestamp + "GET" + "/v1/portfolios/portfolio-123/wallets?limit=10&sort_direction=ASC"
	secretBytes, _ := base64.StdEncoding.DecodeString(secret)
	h := hmac.New(sha256.New, secretBytes)
	h.Write([]byte(message))
	expectedSig := base64.StdEncoding.EncodeToString(h.Sum(nil))

	if sig != expectedSig {
		t.Fatalf("signature mismatch: got %q, want %q", sig, expectedSig)
	}
}

func TestSignRequestIncludesQueryParams(t *testing.T) {
	t.Parallel()

	secret := base64.StdEncoding.EncodeToString([]byte("test-secret"))

	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"balances":[],"pagination":{"next_cursor":"","sort_direction":"ASC","has_next":false}}`))
	}))
	defer server.Close()

	c := NewWithBaseURL("coinbase", "api-key", secret, "passphrase", "portfolio-123", server.URL)

	_, err := c.GetBalances(context.Background(), "cursor-abc", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sig := capturedHeaders.Get("X-CB-ACCESS-SIGNATURE")
	timestamp := capturedHeaders.Get("X-CB-ACCESS-TIMESTAMP")

	// The message should include query params via RequestURI()
	message := timestamp + "GET" + "/v1/portfolios/portfolio-123/balances?cursor=cursor-abc&limit=50&sort_direction=ASC"
	secretBytes, _ := base64.StdEncoding.DecodeString(secret)
	h := hmac.New(sha256.New, secretBytes)
	h.Write([]byte(message))
	expectedSig := base64.StdEncoding.EncodeToString(h.Sum(nil))

	if sig != expectedSig {
		t.Fatalf("signature mismatch (query params not included?): got %q, want %q", sig, expectedSig)
	}
}

func TestSignRequestRejectsInvalidBase64Secret(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("request should not have been sent")
	}))
	defer server.Close()

	c := NewWithBaseURL("coinbase", "api-key", "not-valid-base64!!!", "passphrase", "portfolio-123", server.URL)

	_, err := c.GetWallets(context.Background(), "", 10)
	if err == nil {
		t.Fatal("expected an error for invalid base64 secret")
	}
	if !strings.Contains(err.Error(), "failed to decode API secret") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

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

func TestGetBalancesForSymbolMultiPage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")
		w.WriteHeader(http.StatusOK)

		if cursor == "" {
			// Page 1: only USD
			_, _ = w.Write([]byte(`{
				"balances": [
					{"symbol":"USD","amount":"500.00","holds":"0","withdrawable_amount":"500.00","fiat_amount":"500.00"}
				],
				"pagination": {"next_cursor":"page2","sort_direction":"ASC","has_next":true}
			}`))
		} else if cursor == "page2" {
			// Page 2: only BTC
			_, _ = w.Write([]byte(`{
				"balances": [
					{"symbol":"BTC","amount":"2.0","holds":"0","withdrawable_amount":"2.0","fiat_amount":"120000.00"}
				],
				"pagination": {"next_cursor":"","sort_direction":"ASC","has_next":false}
			}`))
		} else {
			t.Fatalf("unexpected cursor: %s", cursor)
		}
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

func TestGetBalancesForSymbolAggregatesAcrossPages(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")
		w.WriteHeader(http.StatusOK)

		if cursor == "" {
			// Page 1: BTC and USD
			_, _ = w.Write([]byte(`{
				"balances": [
					{"symbol":"BTC","amount":"1.0","holds":"0","withdrawable_amount":"1.0","fiat_amount":"60000.00"},
					{"symbol":"USD","amount":"500.00","holds":"0","withdrawable_amount":"500.00","fiat_amount":"500.00"}
				],
				"pagination": {"next_cursor":"page2","sort_direction":"ASC","has_next":true}
			}`))
		} else if cursor == "page2" {
			// Page 2: BTC and ETH
			_, _ = w.Write([]byte(`{
				"balances": [
					{"symbol":"BTC","amount":"3.0","holds":"1.0","withdrawable_amount":"2.0","fiat_amount":"180000.00"},
					{"symbol":"ETH","amount":"10.0","holds":"0","withdrawable_amount":"10.0","fiat_amount":"30000.00"}
				],
				"pagination": {"next_cursor":"","sort_direction":"ASC","has_next":false}
			}`))
		} else {
			t.Fatalf("unexpected cursor: %s", cursor)
		}
	}))
	defer server.Close()

	c := NewWithBaseURL("coinbase", "api-key", "dGVzdA==", "passphrase", "portfolio-123", server.URL)

	response, err := c.GetBalancesForSymbol(context.Background(), "BTC", "", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Balances) != 2 {
		t.Fatalf("expected 2 balances, got %d", len(response.Balances))
	}

	if response.Balances[0].Amount != "1.0" {
		t.Fatalf("expected first balance amount 1.0, got %s", response.Balances[0].Amount)
	}

	if response.Balances[1].Amount != "3.0" {
		t.Fatalf("expected second balance amount 3.0, got %s", response.Balances[1].Amount)
	}
}
