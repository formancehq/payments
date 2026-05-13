package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Tiny smoke tests so the mock binary stays trustworthy as a fixture.
// The plugin's Ginkgo client tests cover the full surface; here we only
// verify the basics: auth, capabilities, the polling state machine, and
// the webhook subscription idempotency dedup.

func newTestServer(t *testing.T, cfg mockConfig) *httptest.Server {
	t.Helper()
	st := newStore(cfg, nil)
	srv := httptest.NewServer(newServer(cfg, st, nil).Handler())
	t.Cleanup(srv.Close)
	return srv
}

func defaultTestConfig() mockConfig {
	return mockConfig{
		port:             "0",
		apiKey:           "test-key",
		webhookSecret:    "test-secret",
		webhookSignature: "hmac-sha256",
		capabilities:     defaultCapabilities(),
		// evolveOnPoll defaults false in tests so per-test datasets stay
		// deterministic — the suite has dedicated tests below that opt
		// back in to verify poll-driven evolution.
		evolveOnPoll: false,
		evolveBatch:  10,
	}
}

func do(t *testing.T, srv *httptest.Server, method, path string, body any, idemKey string) (*http.Response, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, srv.URL+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-key")
	req.Header.Set("Content-Type", "application/json")
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	return resp, rb
}

// paymentsByRef returns every payment in the mock's /v1/payments list whose
// reference matches `ref`. Used by tests that just want to assert
// "the payout I just initiated is now visible".
func paymentsByRef(t *testing.T, srv *httptest.Server, ref string) []payment {
	t.Helper()
	all := allPayments(t, srv)
	out := all[:0]
	for _, p := range all {
		if p.Reference == ref {
			out = append(out, p)
		}
	}
	return out
}

func allPayments(t *testing.T, srv *httptest.Server) []payment {
	t.Helper()
	_, body := do(t, srv, http.MethodGet, "/v1/payments?pageSize=10000", nil, "")
	var page paymentsPage
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatalf("decode payments: %v", err)
	}
	return page.Items
}

func allOrders(t *testing.T, srv *httptest.Server) []order {
	t.Helper()
	_, body := do(t, srv, http.MethodGet, "/v1/orders?pageSize=10000", nil, "")
	var page ordersPage
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatalf("decode orders: %v", err)
	}
	return page.Items
}

func firstPaymentWithStatus(t *testing.T, srv *httptest.Server, status string) string {
	t.Helper()
	for _, p := range allPayments(t, srv) {
		if p.Status == status {
			return p.Reference
		}
	}
	return ""
}

// Verifies the default mode the user asked for: each poll evolves the
// dataset by `evolveBatch` records when no webhooks are registered.
// Models the 20-minute polling cadence — each poll observes a fresh
// batch of adjustments rather than waiting for a wall-clock ticker.
func TestPollDrivenEvolution_NoWebhooks(t *testing.T) {
	t.Parallel()
	cfg := defaultTestConfig()
	cfg.evolveOnPoll = true
	cfg.evolveBatch = 25
	srv := newTestServer(t, cfg)

	// Each successive poll must reduce the PENDING population — the
	// poll itself drives evolution before serving the response, so we
	// observe progress monotonically.
	prev := -1
	for i := 0; i < 5; i++ {
		got := countWithStatus(t, srv, "PENDING")
		if prev >= 0 && got >= prev {
			t.Fatalf("poll %d: PENDING count did not shrink (prev=%d, now=%d)", i, prev, got)
		}
		prev = got
	}
}

// Verifies the auto-emit path: with a webhook subscription registered,
// /_admin/evolve mutates state AND pushes one matching event per evolved
// record to the registered callback. This is what makes webhook-mode
// useful — without auto-emit, evolutions in webhook-mode wouldn't reach
// the engine until the next 20-min poll.
func TestAutoEmitWebhookOnEvolveDeliversSignedEvents(t *testing.T) {
	t.Parallel()
	cfg := defaultTestConfig()
	srv := newTestServer(t, cfg)

	// Sink that captures every delivery.
	var (
		mu        sync.Mutex
		deliveries []string
	)
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		deliveries = append(deliveries, string(body))
		mu.Unlock()
		// Also assert the signature header is present and decodable.
		if r.Header.Get("X-Universal-Signature") == "" {
			t.Errorf("delivery missing X-Universal-Signature")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer sink.Close()

	// Subscribe to payment.updated.
	_, _ = do(t, srv, http.MethodPost, "/v1/webhooks",
		webhookSubscriptionRequest{Name: "payment.updated", CallbackURL: sink.URL}, "auto-emit-1")

	// Manual evolve — should mutate AND auto-emit one webhook per
	// evolved payment (10 evolves with batch=10 means ~3-4 payments
	// because of the round-robin across payments/orders/conversions).
	resp, body := do(t, srv, http.MethodPost, "/_admin/evolve?n=10", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("evolve: %d %s", resp.StatusCode, string(body))
	}
	var ev map[string]int
	_ = json.Unmarshal(body, &ev)
	if ev["advanced"] == 0 || ev["webhooksDelivered"] == 0 {
		t.Fatalf("evolve response missing counts: %v", ev)
	}

	// Sink should have received at least one POST.
	mu.Lock()
	got := append([]string(nil), deliveries...)
	mu.Unlock()
	if len(got) == 0 {
		t.Fatal("sink received no webhook deliveries")
	}
	if len(got) != ev["webhooksDelivered"] {
		t.Fatalf("delivered count mismatch: response says %d, sink received %d", ev["webhooksDelivered"], len(got))
	}

	// Each delivered body should be a payment.updated event with a real
	// payment resource — exactly what the engine's TranslateWebhook
	// would translate into a PaymentAdjustment.
	var event struct {
		Type     string `json:"type"`
		Resource struct {
			Payment *struct {
				Reference string `json:"reference"`
				Status    string `json:"status"`
			} `json:"payment"`
		} `json:"resource"`
	}
	if err := json.Unmarshal([]byte(got[0]), &event); err != nil {
		t.Fatalf("decode delivery: %v", err)
	}
	if event.Type != "payment.updated" || event.Resource.Payment == nil {
		t.Fatalf("delivered event malformed: %+v", event)
	}
	if event.Resource.Payment.Status == "PENDING" {
		t.Fatalf("auto-emitted payment should reflect post-evolve status, got PENDING")
	}
}

// Verifies that registering a webhook subscription disables poll-driven
// evolution — the engine is now expected to receive state changes via
// pushed events, so polls should be quiet heartbeats.
func TestPollDrivenEvolution_DisabledWhenWebhooksActive(t *testing.T) {
	t.Parallel()
	cfg := defaultTestConfig()
	cfg.evolveOnPoll = true
	cfg.evolveBatch = 50
	srv := newTestServer(t, cfg)

	// Subscribe one webhook — that's enough to flip into "webhook mode".
	_, _ = do(t, srv, http.MethodPost, "/v1/webhooks",
		webhookSubscriptionRequest{Name: "payment.updated", CallbackURL: "http://localhost:9999/x"}, "wh-key-1")

	// Multiple polls in a row should leave the dataset untouched.
	first := countWithStatus(t, srv, "PENDING")
	for i := 0; i < 5; i++ {
		if got := countWithStatus(t, srv, "PENDING"); got != first {
			t.Fatalf("poll %d evolved despite active webhooks: first=%d got=%d", i, first, got)
		}
	}

	// /_admin/evolve remains a manual override even when webhooks gate
	// the poll-driven path.
	resp, _ := do(t, srv, http.MethodPost, "/_admin/evolve?n=10", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin evolve: status=%d", resp.StatusCode)
	}
	if got := countWithStatus(t, srv, "PENDING"); got >= first {
		t.Fatalf("/_admin/evolve should bypass the gate: first=%d got=%d", first, got)
	}
}

func countWithStatus(t *testing.T, srv *httptest.Server, status string) int {
	t.Helper()
	count := 0
	for _, p := range allPayments(t, srv) {
		if p.Status == status {
			count++
		}
	}
	return count
}

// Smoke-tests the auto-evolve goroutine wired up by main(). Pinning the
// interval to 20ms keeps the test fast; we just need to confirm the
// background job actually mutates the store without manual /_admin calls.
func TestAutoEvolveBackgroundTickerDrivesState(t *testing.T) {
	t.Parallel()
	cfg := defaultTestConfig()
	st := newStore(cfg, nil)
	srv := newServer(cfg, st, nil)
	startAutoEvolve(srv, 20*time.Millisecond, 10)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		st.mu.RLock()
		// Look for any payment that has moved off PENDING — proof the
		// ticker has fired and the lane progression is running.
		var moved bool
		for _, p := range st.payments {
			if p.Status != "PENDING" {
				moved = true
				break
			}
		}
		st.mu.RUnlock()
		if moved {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("auto-evolve goroutine never advanced any payment off PENDING within 2s")
}

func TestAuthRequired(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())
	resp, err := http.Get(srv.URL + "/v1/capabilities")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", resp.StatusCode)
	}
}

func TestCapabilities(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())
	resp, body := do(t, srv, http.MethodGet, "/v1/capabilities", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	var got capabilities
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Supported) == 0 {
		t.Fatal("empty supported list")
	}
	if got.Features.WebhookSignature != "hmac-sha256" {
		t.Fatalf("want hmac-sha256 signature, got %q", got.Features.WebhookSignature)
	}
}

func TestSeedFixturesPresent(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())
	expect := map[string]int{
		"/v1/accounts":          seedInternalAccounts,
		"/v1/external-accounts": seedExternalAccounts,
		"/v1/payments":          seedPayments,
		"/v1/orders":            seedOrders,
		"/v1/conversions":       seedConversions,
		"/v1/others/report":     seedOthers,
	}
	for ep, total := range expect {
		ep, total := ep, total
		t.Run(ep, func(t *testing.T) {
			t.Parallel()
			pageSize := 100
			seen := 0
			cursor := ""
			for {
				url := ep + "?pageSize=" + strconv.Itoa(pageSize)
				if cursor != "" {
					url += "&cursor=" + cursor
				}
				resp, body := do(t, srv, http.MethodGet, url, nil, "")
				if resp.StatusCode != http.StatusOK {
					t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
				}
				var page struct {
					Items      []json.RawMessage `json:"items"`
					NextCursor string            `json:"nextCursor"`
					HasMore    bool              `json:"hasMore"`
				}
				if err := json.Unmarshal(body, &page); err != nil {
					t.Fatalf("decode: %v", err)
				}
				seen += len(page.Items)
				if !page.HasMore {
					break
				}
				if page.NextCursor == "" {
					t.Fatalf("hasMore=true but nextCursor empty after %d items", seen)
				}
				cursor = page.NextCursor
			}
			if seen != total {
				t.Fatalf("walked %d items via cursor, want %d", seen, total)
			}
		})
	}
}

func TestPaginationByPageNumber(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())
	// page=2 with pageSize=100 should return rows 100..199 of payments.
	resp, body := do(t, srv, http.MethodGet, "/v1/payments?page=2&pageSize=100", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var page struct {
		Items []struct {
			Reference string `json:"reference"`
		} `json:"items"`
		HasMore bool `json:"hasMore"`
	}
	_ = json.Unmarshal(body, &page)
	if len(page.Items) != 100 {
		t.Fatalf("page 2 returned %d items, want 100", len(page.Items))
	}
	if page.Items[0].Reference != "pay_00100" {
		t.Fatalf("page 2 first ref = %q, want pay_00100", page.Items[0].Reference)
	}
	if !page.HasMore {
		t.Fatal("hasMore should be true on page 2 of 250 records")
	}
}

func TestUpdatedAtFromFiltersForward(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())
	// Fixtures start at 2026-01-01 + index minutes; cut after the 50th payment.
	cut := "2026-01-01T00:50:00Z"
	resp, body := do(t, srv, http.MethodGet, "/v1/payments?pageSize=1000&updatedAtFrom="+cut, nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var page struct {
		Items []json.RawMessage `json:"items"`
	}
	_ = json.Unmarshal(body, &page)
	// Strictly-greater filter: 0..50 excluded, 51..249 kept = 199 items.
	if len(page.Items) != 199 {
		t.Fatalf("updatedAtFrom-filtered count = %d, want 199", len(page.Items))
	}
}

func TestBalancesReturnedPerAccount(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())
	resp, body := do(t, srv, http.MethodGet, "/v1/accounts/acct_internal_000/balances", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), `"accountReference":"acct_internal_000"`) {
		t.Fatalf("balance body missing account ref: %s", body)
	}
}

func TestPayoutTerminalAndPolling(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())

	// Small amount → terminal SUCCEEDED.
	resp, body := do(t, srv, http.MethodPost, "/v1/payouts", initiationRequest{
		Reference: "p1", Amount: "5000", Asset: "EUR/2",
		SourceAccountReference: "acct_internal_eur", DestinationAccountReference: "acct_ext_supplier_a",
	}, "idem-1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	var small initiationResponse
	if err := json.Unmarshal(body, &small); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if small.Mode != "terminal" || small.Payment == nil || small.Payment.Status != "SUCCEEDED" {
		t.Fatalf("want terminal SUCCEEDED, got %+v", small)
	}

	// Large amount → polling, transitions to SUCCEEDED after a few polls.
	resp, body = do(t, srv, http.MethodPost, "/v1/payouts", initiationRequest{
		Reference: "p2", Amount: "999999", Asset: "EUR/2",
		SourceAccountReference: "acct_internal_eur", DestinationAccountReference: "acct_ext_supplier_a",
	}, "idem-2")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	var big initiationResponse
	if err := json.Unmarshal(body, &big); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if big.Mode != "polling" || big.PollingID == "" {
		t.Fatalf("want polling with ID, got %+v", big)
	}

	var lastStatus string
	for i := 0; i < 5; i++ {
		_, body = do(t, srv, http.MethodGet, "/v1/payouts/"+big.PollingID, nil, "")
		var poll initiationResponse
		if err := json.Unmarshal(body, &poll); err != nil {
			t.Fatalf("decode poll %d: %v", i, err)
		}
		if poll.Payment != nil {
			lastStatus = poll.Payment.Status
		}
	}
	if lastStatus != "SUCCEEDED" {
		t.Fatalf("polling never reached SUCCEEDED, last=%q", lastStatus)
	}
}

func TestWebhookSubscriptionIdempotency(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())
	req := webhookSubscriptionRequest{Name: "payment.updated", CallbackURL: "http://localhost:9000/x"}

	_, body1 := do(t, srv, http.MethodPost, "/v1/webhooks", req, "key-1")
	_, body2 := do(t, srv, http.MethodPost, "/v1/webhooks", req, "key-1")

	var r1, r2 webhookSubscriptionResponse
	_ = json.Unmarshal(body1, &r1)
	_ = json.Unmarshal(body2, &r2)
	if r1.ID != r2.ID || r1.ID == "" {
		t.Fatalf("idempotency broken: %q vs %q", r1.ID, r2.ID)
	}
}

func TestWebhookSubscriptionDelete(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())
	resp, _ := do(t, srv, http.MethodPost, "/v1/webhooks",
		webhookSubscriptionRequest{Name: "payment.updated", CallbackURL: "http://localhost:9000/x"}, "key-1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: %d", resp.StatusCode)
	}
	resp, _ = do(t, srv, http.MethodDelete, "/v1/webhooks/sub_payment.updated", nil, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: %d", resp.StatusCode)
	}
}

func TestTransferTerminalAndPolling(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())

	// Transfers and payouts MUST track separate idempotency state — same
	// idem key under different primitives doesn't collide.
	resp, body := do(t, srv, http.MethodPost, "/v1/transfers", initiationRequest{
		Reference: "tx1", Amount: "5000", Asset: "EUR/2",
		SourceAccountReference: "acct_internal_000", DestinationAccountReference: "acct_internal_001",
	}, "idem-tx-1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	var small initiationResponse
	_ = json.Unmarshal(body, &small)
	if small.Mode != "terminal" || small.Payment == nil || small.Payment.Type != "TRANSFER" || small.Payment.Status != "SUCCEEDED" {
		t.Fatalf("want terminal SUCCEEDED TRANSFER, got %+v / payment=%+v", small, small.Payment)
	}

	resp, body = do(t, srv, http.MethodPost, "/v1/transfers", initiationRequest{
		Reference: "tx2", Amount: "999999", Asset: "EUR/2",
		SourceAccountReference: "acct_internal_000", DestinationAccountReference: "acct_internal_001",
	}, "idem-tx-2")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	var big initiationResponse
	_ = json.Unmarshal(body, &big)
	if big.Mode != "polling" || big.PollingID == "" {
		t.Fatalf("want polling, got %+v", big)
	}
	if !strings.HasPrefix(big.PollingID, "ptransfer_") {
		t.Fatalf("transfer polling ID should be namespaced, got %q", big.PollingID)
	}

	var lastStatus, lastType string
	for i := 0; i < 5; i++ {
		_, body = do(t, srv, http.MethodGet, "/v1/transfers/"+big.PollingID, nil, "")
		var poll initiationResponse
		_ = json.Unmarshal(body, &poll)
		if poll.Payment != nil {
			lastStatus, lastType = poll.Payment.Status, poll.Payment.Type
		}
	}
	if lastStatus != "SUCCEEDED" || lastType != "TRANSFER" {
		t.Fatalf("transfer poll never reached terminal SUCCEEDED TRANSFER (status=%q type=%q)", lastStatus, lastType)
	}
}

func TestReverseEndpointsReturnRefundedPayment(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())
	for _, ep := range []struct{ url, kind string }{
		{"/v1/payouts/anyref/reverse", "PAYOUT"},
		{"/v1/transfers/anyref/reverse", "TRANSFER"},
	} {
		resp, body := do(t, srv, http.MethodPost, ep.url, initiationRequest{
			Reference: "rev-1", Amount: "1000", Asset: "EUR/2",
		}, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: status=%d body=%s", ep.url, resp.StatusCode, string(body))
		}
		var r initiationResponse
		_ = json.Unmarshal(body, &r)
		if r.Payment == nil || r.Payment.Status != "REFUNDED" || r.Payment.Type != ep.kind {
			t.Fatalf("%s: want REFUNDED %s, got %+v", ep.url, ep.kind, r.Payment)
		}
	}
}

func TestCreateBankAccountIdempotencyAndExternalAccountSurface(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())
	iban := "GB29NWBK60161331926819"
	req := bankAccountRequest{
		ID:        "ba-uuid-1",
		CreatedAt: time.Now().UTC(),
		Name:      "Treasury GBP",
		IBAN:      &iban,
	}
	_, body1 := do(t, srv, http.MethodPost, "/v1/bank-accounts", req, "ba-idem-1")
	var r1 bankAccountResponse
	_ = json.Unmarshal(body1, &r1)
	if r1.RelatedAccount.Reference == "" {
		t.Fatal("first POST: empty related account reference")
	}
	if r1.RelatedAccount.DefaultAsset == nil || *r1.RelatedAccount.DefaultAsset != "GBP/2" {
		t.Fatalf("GB IBAN should map to GBP/2, got %v", r1.RelatedAccount.DefaultAsset)
	}

	// Second POST with same idem key returns the same account ref (no
	// duplicate appended to the external-accounts list).
	_, body2 := do(t, srv, http.MethodPost, "/v1/bank-accounts", req, "ba-idem-1")
	var r2 bankAccountResponse
	_ = json.Unmarshal(body2, &r2)
	if r2.RelatedAccount.Reference != r1.RelatedAccount.Reference {
		t.Fatalf("idempotent reuse broken: %q vs %q", r2.RelatedAccount.Reference, r1.RelatedAccount.Reference)
	}

	// Subsequent FetchExternalAccounts surfaces the created account.
	_, body3 := do(t, srv, http.MethodGet, "/v1/external-accounts?pageSize=1000", nil, "")
	if !strings.Contains(string(body3), r1.RelatedAccount.Reference) {
		t.Fatalf("created bank account %q not surfaced in /v1/external-accounts", r1.RelatedAccount.Reference)
	}
}

// Verifies that POST /v1/payouts populates /v1/payments with a Payment
// whose Reference matches the engine-side initiation reference, so the
// engine can correlate PaymentInitiationAdjustment ↔ PaymentAdjustment.
func TestPayoutSurfacesInPaymentsList(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())

	// Polling payout (amount > €100) — should appear immediately as PENDING.
	_, _ = do(t, srv, http.MethodPost, "/v1/payouts", initiationRequest{
		Reference: "po-vis-1", Amount: "500000", Asset: "EUR/2",
		SourceAccountReference: "acct_internal_000", DestinationAccountReference: "acct_ext_000",
	}, "idem-vis-1")

	// Filter to only this reference to keep the assertion tight.
	payments := paymentsByRef(t, srv, "po-vis-1")
	if len(payments) != 1 {
		t.Fatalf("polling payout: want 1 matching payment, got %d", len(payments))
	}
	if payments[0].Status != "PENDING" || payments[0].Type != "PAYOUT" {
		t.Fatalf("polling payout: want PENDING PAYOUT, got %+v", payments[0])
	}

	// Drive the poll loop to completion. Each pollPayout call advances the
	// state machine and upserts the payment.
	for i := 0; i < 5; i++ {
		_, _ = do(t, srv, http.MethodGet, "/v1/payouts/ppayout_po-vis-1", nil, "")
	}

	payments = paymentsByRef(t, srv, "po-vis-1")
	if len(payments) != 1 {
		t.Fatalf("after polling: want 1 payment, got %d", len(payments))
	}
	if payments[0].Status != "SUCCEEDED" {
		t.Fatalf("after polling: want SUCCEEDED, got %s", payments[0].Status)
	}
}

// Verifies that /_admin/evolve advances payment status and bumps
// updatedAt, so subsequent FetchNextPayments calls observe the change
// (which the engine then turns into a fresh PaymentAdjustment).
func TestEvolveAdvancesPaymentStateAndBumpsUpdatedAt(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())

	// Find a PENDING payment to track.
	target := firstPaymentWithStatus(t, srv, "PENDING")
	if target == "" {
		t.Fatal("no PENDING payment in seed data")
	}

	before := paymentsByRef(t, srv, target)[0]

	resp, body := do(t, srv, http.MethodPost, "/_admin/evolve?n=100", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("evolve: %d %s", resp.StatusCode, string(body))
	}
	var ev map[string]int
	_ = json.Unmarshal(body, &ev)
	if ev["advanced"] == 0 {
		t.Fatal("expected at least 1 advance")
	}

	after := paymentsByRef(t, srv, target)[0]
	if after.Status == before.Status {
		t.Fatalf("status unchanged after evolve: %s", after.Status)
	}
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Fatalf("updatedAt did not advance: before=%s after=%s", before.UpdatedAt, after.UpdatedAt)
	}
}

// Verifies the lane catalogue actually drives every adjustment path
// observable by the engine. We mimic the engine's poll-evolve-poll loop:
// at each iteration, evolve N records and snapshot every payment + order
// status. The union over all snapshots is exactly what the engine's
// PaymentAdjustment / OrderAdjustment derivation would observe.
func TestEvolutionDrivesEveryAdjustmentPath(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())

	paymentStates := map[string]struct{}{}
	orderStates := map[string]struct{}{}

	for i := 0; i < 100; i++ {
		_, _ = do(t, srv, http.MethodPost, "/_admin/evolve?n=40", nil, "")

		for _, p := range allPayments(t, srv) {
			paymentStates[p.Status] = struct{}{}
		}
		for _, o := range allOrders(t, srv) {
			ordered := mustParseInt(o.BaseQuantityOrdered)
			filled := mustParseInt(o.BaseQuantityFilled)
			pct := 0
			if ordered > 0 {
				pct = int(filled * 100 / ordered)
			}
			orderStates[o.Status+"@"+strconv.Itoa(pct)] = struct{}{}
		}
	}

	// Every entry below corresponds to a different mapping in
	// FromPaymentDataToPaymentInitiationAdjustment — seeing all of them
	// means every adjustment-status path the engine can derive is
	// exercised by the mock.
	for _, want := range []string{
		"SUCCEEDED", "FAILED", "CANCELLED", "EXPIRED",
		"REFUNDED", "REFUND_REVERSED", "REFUNDED_FAILURE",
		"DISPUTE_WON", "DISPUTE_LOST", "CAPTURE_FAILED",
	} {
		if _, ok := paymentStates[want]; !ok {
			t.Errorf("payment status %q never observed; saw %v", want, sortedKeys(paymentStates))
		}
	}

	// Order partial-fill progression: the engine's OrderAdjustment dedup
	// keys on BaseQuantityFilled, so each PARTIALLY_FILLED@N and the
	// terminal FILLED/CANCELLED/EXPIRED states must all show up.
	for _, want := range []string{
		"PARTIALLY_FILLED@25", "PARTIALLY_FILLED@50", "PARTIALLY_FILLED@75",
		"FILLED@100", "CANCELLED@33", "EXPIRED@0", "FAILED@0",
	} {
		if _, ok := orderStates[want]; !ok {
			t.Errorf("order state %q never observed; saw %v", want, sortedKeys(orderStates))
		}
	}
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Verifies the admin-trigger path materialises a real resource the engine
// can deserialize, signs it, and delivers it. We capture the delivery on
// a sink httptest.Server, decode the payload, and assert the resource
// shape matches the event type.
func TestAdminTriggerDeliversSignedPayload(t *testing.T) {
	t.Parallel()
	cfg := defaultTestConfig()

	// Set up a sink to receive the delivered webhook.
	var captured []byte
	var capturedSig, capturedTs string
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured, _ = io.ReadAll(r.Body)
		capturedSig = r.Header.Get("X-Universal-Signature")
		capturedTs = r.Header.Get("X-Universal-Timestamp")
		w.WriteHeader(http.StatusOK)
	}))
	defer sink.Close()

	srv := newTestServer(t, cfg)
	// Subscribe.
	_, _ = do(t, srv, http.MethodPost, "/v1/webhooks",
		webhookSubscriptionRequest{Name: "payment.updated", CallbackURL: sink.URL}, "trig-1")
	// Trigger.
	resp, body := do(t, srv, http.MethodPost, "/_admin/trigger-webhook?name=payment.updated", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("trigger: status=%d body=%s", resp.StatusCode, string(body))
	}
	if len(captured) == 0 {
		t.Fatal("sink did not receive a delivery")
	}
	if capturedSig == "" || capturedTs == "" {
		t.Fatal("delivery missing signature or timestamp headers")
	}
	expectedSig := signHMAC(cfg.webhookSecret, capturedTs, captured)
	if expectedSig != capturedSig {
		t.Fatalf("signature mismatch:\nwant=%s\ngot =%s", expectedSig, capturedSig)
	}
	// Decode and assert resource has `payment.reference`.
	var event struct {
		Type     string `json:"type"`
		Resource struct {
			Payment *struct {
				Reference string `json:"reference"`
				Status    string `json:"status"`
			} `json:"payment"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(captured, &event); err != nil {
		t.Fatalf("decode delivered body: %v", err)
	}
	if event.Type != "payment.updated" {
		t.Fatalf("event.type=%q want payment.updated", event.Type)
	}
	if event.Resource.Payment == nil || event.Resource.Payment.Reference == "" {
		t.Fatalf("payment resource missing or empty: %+v", event.Resource)
	}
}
