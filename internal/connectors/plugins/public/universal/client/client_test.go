package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// httptest-backed sanity check: the client builds requests properly,
// passes the bearer token, sends the Idempotency-Key on POSTs, and decodes
// pages from the contract correctly.

func TestClient_GetCapabilities(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/capabilities" || r.Method != http.MethodGet {
			t.Fatalf("unexpected req: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer abc" {
			t.Fatalf("auth header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(CapabilitiesResponse{
			Supported: []string{"FETCH_ACCOUNTS"},
			Features:  Features{Pagination: "cursor"},
		})
	}))
	defer srv.Close()

	c := New("universal-test", srv.URL, "abc")
	res, err := c.GetCapabilities(context.Background())
	if err != nil {
		t.Fatalf("GetCapabilities: %v", err)
	}
	if len(res.Supported) != 1 || res.Supported[0] != "FETCH_ACCOUNTS" {
		t.Fatalf("unexpected supported: %v", res.Supported)
	}
}

func TestClient_CreatePayoutSendsIdempotencyKey(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(IdempotencyHeader) != "ref-1" {
			t.Fatalf("missing or wrong idempotency key: %q", r.Header.Get(IdempotencyHeader))
		}
		_ = json.NewEncoder(w).Encode(PayoutResponse{Mode: "terminal"})
	}))
	defer srv.Close()

	c := New("universal-test", srv.URL, "abc")
	_, err := c.CreatePayout(context.Background(), "ref-1", &PayoutRequest{
		Reference: "ref-1", Amount: "100", Asset: "EUR/2",
		SourceAccountReference: "s", DestinationAccountReference: "d",
	})
	if err != nil {
		t.Fatalf("CreatePayout: %v", err)
	}
}

func TestClient_PropagatesErrorEnvelope(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"title":"Validation","detail":"amount missing"}`))
	}))
	defer srv.Close()

	c := New("universal-test", srv.URL, "abc")
	_, err := c.GetCapabilities(context.Background())
	if err == nil {
		t.Fatal("want error")
	}
	if !strings.Contains(err.Error(), "Validation") || !strings.Contains(err.Error(), "amount missing") {
		t.Fatalf("error text didn't propagate envelope: %v", err)
	}
}
