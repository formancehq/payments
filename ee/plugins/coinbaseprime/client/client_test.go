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

func TestSignRequestUsesRawStringSecret(t *testing.T) {
	t.Parallel()

	// Coinbase Prime signing keys are used as raw string bytes, not base64-decoded.
	// This is a fake key that mimics the Coinbase Prime format (with = in the middle).
	secret := "fAkEsEcReT4qM=xYzAbCdEfGhIjKlMnOpQrStUvWxYz0123456789+/ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789+/abcd=="

	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"wallets":[],"pagination":{"next_cursor":"","sort_direction":"ASC","has_next":false}}`))
	}))
	defer server.Close()

	c := NewWithBaseURL("coinbaseprime", "api-key", secret, "passphrase", "portfolio-123", server.URL)

	_, err := c.GetWallets(context.Background(), "", "", 10)
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

	// Verify the signature was computed with the raw string bytes (not base64-decoded)
	// and uses only the path (no query params).
	message := timestamp + "GET" + "/v1/portfolios/portfolio-123/wallets"
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	expectedSig := base64.StdEncoding.EncodeToString(h.Sum(nil))

	if sig != expectedSig {
		t.Fatalf("signature mismatch: got %q, want %q", sig, expectedSig)
	}
}

func TestSignRequestExcludesQueryParams(t *testing.T) {
	t.Parallel()

	secret := "test-signing-key"

	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"wallets":[],"pagination":{"next_cursor":"","sort_direction":"ASC","has_next":false}}`))
	}))
	defer server.Close()

	c := NewWithBaseURL("coinbaseprime", "api-key", secret, "passphrase", "portfolio-123", server.URL)

	_, err := c.GetWallets(context.Background(), "", "cursor-abc", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sig := capturedHeaders.Get("X-CB-ACCESS-SIGNATURE")
	timestamp := capturedHeaders.Get("X-CB-ACCESS-TIMESTAMP")

	// The signature message must use only the path, NOT query params.
	message := timestamp + "GET" + "/v1/portfolios/portfolio-123/wallets"
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	expectedSig := base64.StdEncoding.EncodeToString(h.Sum(nil))

	if sig != expectedSig {
		t.Fatalf("signature mismatch (query params should NOT be included): got %q, want %q", sig, expectedSig)
	}
}

func TestSignRequestSetsAllRequiredHeaders(t *testing.T) {
	t.Parallel()

	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"wallets":[],"pagination":{"next_cursor":"","sort_direction":"ASC","has_next":false}}`))
	}))
	defer server.Close()

	c := NewWithBaseURL("coinbaseprime", "my-access-key", "my-signing-key", "my-passphrase", "portfolio-123", server.URL)

	_, err := c.GetWallets(context.Background(), "", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := capturedHeaders.Get("X-CB-ACCESS-KEY"); got != "my-access-key" {
		t.Fatalf("expected X-CB-ACCESS-KEY to be %q, got %q", "my-access-key", got)
	}
	if got := capturedHeaders.Get("X-CB-ACCESS-PASSPHRASE"); got != "my-passphrase" {
		t.Fatalf("expected X-CB-ACCESS-PASSPHRASE to be %q, got %q", "my-passphrase", got)
	}
	if capturedHeaders.Get("X-CB-ACCESS-SIGNATURE") == "" {
		t.Fatal("expected X-CB-ACCESS-SIGNATURE header to be set")
	}
	if capturedHeaders.Get("X-CB-ACCESS-TIMESTAMP") == "" {
		t.Fatal("expected X-CB-ACCESS-TIMESTAMP header to be set")
	}
	if got := capturedHeaders.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type to be application/json, got %q", got)
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
		case "/v1/portfolios/" + portfolioID + "/transactions":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"transactions":[],"pagination":{"next_cursor":"","sort_direction":"ASC","has_next":false}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewWithBaseURL("coinbaseprime", "api-key", "signing-key", "passphrase", portfolioID, server.URL)

	if _, err := c.GetWallets(context.Background(), "", cursor, pageSize); err != nil {
		t.Fatalf("GetWallets failed: %v", err)
	}
	if _, err := c.GetTransactions(context.Background(), cursor, pageSize); err != nil {
		t.Fatalf("GetTransactions failed: %v", err)
	}
}

