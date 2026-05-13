package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// One http handler per route exercises the full operations.go pass-through
// surface. Each subtest verifies the method + path the client sends and
// returns a minimal valid envelope so we can assert decoding works.

func TestClient_AllOperations(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/accounts":
			_ = json.NewEncoder(w).Encode(AccountsPage{HasMore: false})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/external-accounts":
			_ = json.NewEncoder(w).Encode(AccountsPage{HasMore: false})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/accounts/a1/balances":
			_ = json.NewEncoder(w).Encode(BalancesResponse{})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/payments":
			_ = json.NewEncoder(w).Encode(PaymentsPage{HasMore: false})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/orders":
			_ = json.NewEncoder(w).Encode(OrdersPage{HasMore: false})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/conversions":
			_ = json.NewEncoder(w).Encode(ConversionsPage{HasMore: false})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/others/report":
			_ = json.NewEncoder(w).Encode(OthersPage{HasMore: false})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/payouts/x":
			_ = json.NewEncoder(w).Encode(PayoutResponse{Mode: "polling"})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/payouts/x/reverse":
			_ = json.NewEncoder(w).Encode(PayoutResponse{Mode: "terminal"})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/transfers":
			_ = json.NewEncoder(w).Encode(TransferResponse{Mode: "terminal"})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/transfers/y":
			_ = json.NewEncoder(w).Encode(TransferResponse{Mode: "polling"})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/transfers/y/reverse":
			_ = json.NewEncoder(w).Encode(TransferResponse{Mode: "terminal"})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/bank-accounts":
			_ = json.NewEncoder(w).Encode(BankAccountResponse{})
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/webhooks/sub-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New("universal-test", srv.URL, "k")
	ctx := context.Background()
	pg := Pagination{PageSize: 10}

	if _, err := c.ListAccounts(ctx, pg); err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if _, err := c.ListExternalAccounts(ctx, pg); err != nil {
		t.Fatalf("ListExternalAccounts: %v", err)
	}
	if _, err := c.GetBalances(ctx, "a1"); err != nil {
		t.Fatalf("GetBalances: %v", err)
	}
	if _, err := c.ListPayments(ctx, pg); err != nil {
		t.Fatalf("ListPayments: %v", err)
	}
	if _, err := c.ListOrders(ctx, pg); err != nil {
		t.Fatalf("ListOrders: %v", err)
	}
	if _, err := c.ListConversions(ctx, pg); err != nil {
		t.Fatalf("ListConversions: %v", err)
	}
	if _, err := c.ListOthers(ctx, "report", pg); err != nil {
		t.Fatalf("ListOthers: %v", err)
	}
	if _, err := c.GetPayout(ctx, "x"); err != nil {
		t.Fatalf("GetPayout: %v", err)
	}
	if _, err := c.ReversePayout(ctx, "k", "x", &ReverseRequest{}); err != nil {
		t.Fatalf("ReversePayout: %v", err)
	}
	if _, err := c.CreateTransfer(ctx, "k", &TransferRequest{}); err != nil {
		t.Fatalf("CreateTransfer: %v", err)
	}
	if _, err := c.GetTransfer(ctx, "y"); err != nil {
		t.Fatalf("GetTransfer: %v", err)
	}
	if _, err := c.ReverseTransfer(ctx, "k", "y", &ReverseRequest{}); err != nil {
		t.Fatalf("ReverseTransfer: %v", err)
	}
	if _, err := c.CreateBankAccount(ctx, "k", &BankAccountRequest{}); err != nil {
		t.Fatalf("CreateBankAccount: %v", err)
	}
	if err := c.DeleteWebhookSubscription(ctx, "sub-1"); err != nil {
		t.Fatalf("DeleteWebhookSubscription: %v", err)
	}
}

func TestClient_SetIdempotencyHeaderOverridesAndAlwaysIncludesCanonical(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Both the override and the canonical header are set.
		if got := r.Header.Get("X-Token"); got != "ref-1" {
			t.Errorf("X-Token = %q; want ref-1", got)
		}
		if got := r.Header.Get(IdempotencyHeader); got != "ref-1" {
			t.Errorf("canonical Idempotency-Key = %q; want ref-1", got)
		}
		_ = json.NewEncoder(w).Encode(PayoutResponse{Mode: "terminal"})
	}))
	defer srv.Close()

	c := New("universal-test", srv.URL, "k")
	c.SetIdempotencyHeader("X-Token")

	if _, err := c.CreatePayout(context.Background(), "ref-1", &PayoutRequest{
		Reference: "ref-1", Amount: "1", Asset: "EUR/2",
		SourceAccountReference: "s", DestinationAccountReference: "d",
	}); err != nil {
		t.Fatalf("CreatePayout: %v", err)
	}
}

func TestClient_AddPaginationEncodesAllKnobs(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		for _, key := range []string{"cursor", "page", "pageSize", "updatedAtFrom"} {
			if q.Get(key) == "" {
				t.Errorf("missing query parameter %q in %q", key, r.URL.RawQuery)
			}
		}
		_ = json.NewEncoder(w).Encode(AccountsPage{})
	}))
	defer srv.Close()

	ts, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	c := New("universal-test", srv.URL, "k").(*client)
	url := c.addPagination(c.url("/v1/accounts"), Pagination{
		Cursor: "abc", PageNumber: 2, PageSize: 50, UpdatedAtFrom: ts,
	})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if _, err := http.DefaultClient.Do(req); err != nil {
		t.Fatalf("get: %v", err)
	}
}
