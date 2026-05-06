package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestServer spins up a Routable-shaped httptest server that asserts the
// authentication header the client must send and routes requests through a
// per-test handler. Returning a fresh Client wired to the test server keeps
// each test isolated.
func newTestServer(t *testing.T, handler http.HandlerFunc) (Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("expected Bearer auth header, got %q", got)
		}
		handler(w, r)
	}))
	return New("routable-test", "test-key", srv.URL), srv
}

func TestListAccounts(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/settings/accounts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "1" || r.URL.Query().Get("page_size") != "25" {
			t.Errorf("unexpected pagination: %s", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, `{"object":"List","results":[{"id":"acc_1","name":"Operating","type_details":{"available_amount":"100.00"},"created_at":"2025-01-01T00:00:00Z"}],"links":{"next":"/v1/settings/accounts?page=2"}}`)
	})
	defer srv.Close()

	resp, err := c.ListAccounts(context.Background(), 1, 25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].ID != "acc_1" {
		t.Fatalf("unexpected results: %+v", resp.Results)
	}
	if !resp.Links.HasMore() {
		t.Fatalf("expected HasMore=true")
	}
}

func TestGetPayableNotFound(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"object":"Error","message":"not found"}`)
	})
	defer srv.Close()

	_, err := c.GetPayable(context.Background(), "pa_missing")
	if !errors.Is(err, ErrPayableNotFound) {
		t.Fatalf("expected ErrPayableNotFound, got %v", err)
	}
}

func TestCreatePayableSendsIdempotencyKey(t *testing.T) {
	var captured struct {
		Key  string
		Body CreatePayableRequest
	}
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		captured.Key = r.Header.Get("Idempotency-Key")
		_ = json.NewDecoder(r.Body).Decode(&captured.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"object":"Payable","id":"pa_1","status":"pending","amount":"10.00","currency_code":"USD","created_at":"2025-01-01T00:00:00Z"}`)
	})
	defer srv.Close()

	req := CreatePayableRequest{
		Type:                "ach",
		DeliveryMethod:      "ach_standard",
		PayToCompany:        "co_1",
		WithdrawFromAccount: "acc_1",
		Amount:              "10.00",
		CurrencyCode:        "USD",
		LineItems:           []PayableLineItem{{UnitPrice: "10.00", Amount: "10.00", Quantity: 1}},
		ActingTeamMember:    "tm_1",
		IdempotencyKey:      "pi_42",
	}
	resp, err := c.CreatePayable(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "pa_1" {
		t.Fatalf("unexpected payable id: %s", resp.ID)
	}
	if captured.Key != "pi_42" {
		t.Fatalf("expected idempotency key 'pi_42', got %q", captured.Key)
	}
	if captured.Body.PayToCompany != "co_1" {
		t.Fatalf("body not forwarded: %+v", captured.Body)
	}
}

func TestCreatePayableValidatesInputBeforeNetwork(t *testing.T) {
	c := New("routable-test", "test-key", "http://invalid")
	if _, err := c.CreatePayable(context.Background(), CreatePayableRequest{Type: ""}); err == nil {
		t.Fatal("expected validation error before any network call")
	}
}

func TestErrorSuffixOnlyWhenAPIBodyPresent(t *testing.T) {
	t.Parallel()

	t.Run("with API body", func(t *testing.T) {
		c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"object":"Error","code":"invalid","message":"bad amount"}`)
		})
		defer srv.Close()

		_, err := c.ListAccounts(context.Background(), 1, 25)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "bad amount") {
			t.Fatalf("expected error to surface API message, got %q", err.Error())
		}
	})

	t.Run("without API body", func(t *testing.T) {
		c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			// no JSON body
		})
		defer srv.Close()

		_, err := c.ListAccounts(context.Background(), 1, 25)
		if err == nil {
			t.Fatal("expected error")
		}
		// We must not pollute the message with the placeholder "empty body" suffix.
		if strings.Contains(err.Error(), "routable api error: empty body") {
			t.Fatalf("error should not include placeholder suffix, got %q", err.Error())
		}
	})
}

func TestListPayablesPassesStatusChangedAtGte(t *testing.T) {
	when := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		got := r.URL.Query().Get("status_changed_at.gte")
		if !strings.HasPrefix(got, "2025-01-02T03:04:05") {
			t.Errorf("missing/bad status_changed_at.gte: %q", got)
		}
		_, _ = io.WriteString(w, `{"object":"List","results":[]}`)
	})
	defer srv.Close()

	if _, err := c.ListPayables(context.Background(), 1, 50, when); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
