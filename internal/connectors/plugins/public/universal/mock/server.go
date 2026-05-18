package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// webhookHTTP is the dedicated client for outbound webhook deliveries.
// Bounded timeout so a stalled consumer can't tie up the mock for the
// connect timeout's eternity.
var webhookHTTP = &http.Client{Timeout: 10 * time.Second}

// server fronts the contract endpoints with bearer-auth on every route
// except /healthz and /v1/stream (auth happens inside the WS handshake).
type server struct {
	cfg    mockConfig
	store  *store
	mux    *http.ServeMux
	logger *slog.Logger
	hub    *streamHub
}

// newServer returns a wired server. Use Handler() to get the
// auth+logging-wrapped http.Handler. Tests construct via newServer to
// reach internal helpers (evolveAndDeliver, …) directly.
func newServer(cfg mockConfig, st *store, logger *slog.Logger) *server {
	if logger == nil {
		logger = newLogger("info")
	}
	s := &server{
		cfg:    cfg,
		store:  st,
		mux:    http.NewServeMux(),
		logger: logger,
		hub:    newStreamHub(logger),
	}
	s.routes()
	return s
}

// Handler — outside-in: logging → auth → mux. Unauth requests still
// produce a response log line.
func (s *server) Handler() http.Handler {
	return s.loggingMiddleware(s.requireAuth(s.mux))
}

func (s *server) routes() {
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	s.mux.HandleFunc("GET /v1/capabilities", s.handleCapabilities)

	s.mux.HandleFunc("GET /v1/accounts", s.handleListAccounts)
	s.mux.HandleFunc("GET /v1/external-accounts", s.handleListExternalAccounts)
	s.mux.HandleFunc("GET /v1/accounts/{id}/balances", s.handleAccountBalances)

	s.mux.HandleFunc("GET /v1/payments", s.handleListPayments)
	s.mux.HandleFunc("GET /v1/orders", s.handleListOrders)
	s.mux.HandleFunc("GET /v1/conversions", s.handleListConversions)
	s.mux.HandleFunc("GET /v1/others/{name}", s.handleListOthers)

	s.mux.HandleFunc("POST /v1/payouts", s.handleCreatePayout)
	s.mux.HandleFunc("GET /v1/payouts/{id}", s.handleGetPayout)
	s.mux.HandleFunc("POST /v1/payouts/{id}/reverse", s.handleReversePayout)

	s.mux.HandleFunc("POST /v1/transfers", s.handleCreateTransfer)
	s.mux.HandleFunc("GET /v1/transfers/{id}", s.handleGetTransfer)
	s.mux.HandleFunc("POST /v1/transfers/{id}/reverse", s.handleReverseTransfer)

	s.mux.HandleFunc("POST /v1/bank-accounts", s.handleCreateBankAccount)

	s.mux.HandleFunc("POST /v1/webhooks", s.handleCreateWebhookSub)
	s.mux.HandleFunc("DELETE /v1/webhooks/{id}", s.handleDeleteWebhookSub)

	// WebSocket stream (optional, gated on MOCK_EVENT_STREAM=wss).
	// Auth happens inside the handshake — the requireAuth middleware
	// is bypassed because browsers can't set Authorization on WS, so
	// we accept either the header (when present) or apiKey in the
	// signed hello.
	s.mux.HandleFunc("GET /v1/stream", s.handleStream)

	// Out-of-contract admin endpoints — see mock/README.md.
	s.mux.HandleFunc("POST /_admin/trigger-webhook", s.handleAdminTrigger)
	s.mux.HandleFunc("POST /_admin/evolve", s.handleAdminEvolve)
}

// requireAuth — bearer-auth middleware. Token is never logged; outcome
// is. /healthz and /v1/stream bypass: stream auth happens inside the
// signed handshake instead (browser WS can't set headers).
func (s *server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/v1/stream" {
			next.ServeHTTP(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != s.cfg.apiKey {
			reqLogger(r.Context(), s.logger).Warn("auth rejected",
				"have_header", auth != "",
				"scheme_ok", strings.HasPrefix(auth, "Bearer "),
			)
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	// High-signal: "someone is installing the connector".
	reqLogger(r.Context(), s.logger).Info("capabilities discovery",
		"supported_count", len(s.cfg.capabilities),
		"signature_scheme", s.cfg.webhookSignature,
		"event_stream", s.cfg.eventStream,
	)
	writeJSON(w, http.StatusOK, capabilities{
		Supported: s.cfg.capabilities,
		Features: features{
			Pagination:       "cursor",
			WebhookSignature: s.cfg.webhookSignature,
			EventStream:      s.cfg.eventStream,
			StreamEvents:     s.cfg.streamEvents,
		},
	})
}

func (s *server) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	s.maybeEvolveOnPoll(r.Context())
	items, next, hasMore := s.store.listAccounts(parseListOpts(r))
	reqLogger(r.Context(), s.logger).Debug("listed accounts", "count", len(items), "has_more", hasMore)
	writeJSON(w, http.StatusOK, accountsPage{Items: items, NextCursor: next, HasMore: hasMore})
}

func (s *server) handleListExternalAccounts(w http.ResponseWriter, r *http.Request) {
	s.maybeEvolveOnPoll(r.Context())
	items, next, hasMore := s.store.listExternalAccounts(parseListOpts(r))
	reqLogger(r.Context(), s.logger).Debug("listed external accounts", "count", len(items), "has_more", hasMore)
	writeJSON(w, http.StatusOK, accountsPage{Items: items, NextCursor: next, HasMore: hasMore})
}

func (s *server) handleAccountBalances(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	bals := s.store.accountBalances(id)
	reqLogger(r.Context(), s.logger).Debug("listed balances", "account", id, "count", len(bals))
	writeJSON(w, http.StatusOK, balancesResponse{Items: bals})
}

func (s *server) handleListPayments(w http.ResponseWriter, r *http.Request) {
	s.maybeEvolveOnPoll(r.Context())
	items, next, hasMore := s.store.listPayments(parseListOpts(r))
	reqLogger(r.Context(), s.logger).Debug("listed payments", "count", len(items), "has_more", hasMore)
	writeJSON(w, http.StatusOK, paymentsPage{Items: items, NextCursor: next, HasMore: hasMore})
}

func (s *server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	s.maybeEvolveOnPoll(r.Context())
	items, next, hasMore := s.store.listOrders(parseListOpts(r))
	reqLogger(r.Context(), s.logger).Debug("listed orders", "count", len(items), "has_more", hasMore)
	writeJSON(w, http.StatusOK, ordersPage{Items: items, NextCursor: next, HasMore: hasMore})
}

func (s *server) handleListConversions(w http.ResponseWriter, r *http.Request) {
	s.maybeEvolveOnPoll(r.Context())
	items, next, hasMore := s.store.listConversions(parseListOpts(r))
	reqLogger(r.Context(), s.logger).Debug("listed conversions", "count", len(items), "has_more", hasMore)
	writeJSON(w, http.StatusOK, conversionsPage{Items: items, NextCursor: next, HasMore: hasMore})
}

func (s *server) handleListOthers(w http.ResponseWriter, r *http.Request) {
	s.maybeEvolveOnPoll(r.Context())
	name := r.PathValue("name")
	items, next, hasMore := s.store.listOthers(name, parseListOpts(r))
	reqLogger(r.Context(), s.logger).Debug("listed others", "name", name, "count", len(items), "has_more", hasMore)
	writeJSON(w, http.StatusOK, othersPage{Items: items, NextCursor: next, HasMore: hasMore})
}

// maybeEvolveOnPoll advances `evolveBatch` records when poll-driven
// evolution is on AND no webhooks are registered. Evolution happens
// before the response so the same poll observes the new state.
func (s *server) maybeEvolveOnPoll(ctx context.Context) {
	if !s.cfg.evolveOnPoll {
		return
	}
	if s.store.webhooksRegistered() {
		reqLogger(ctx, s.logger).Debug("poll-driven evolution skipped (webhooks active)")
		return
	}
	results := s.store.EvolveSteps(s.cfg.evolveBatch)
	if len(results) > 0 {
		reqLogger(ctx, s.logger).Info("poll-driven evolution",
			"records_advanced", len(results),
			"first_ref", results[0].Reference,
		)
	}
}

// evolveAndDeliver advances `n` records and pushes each transition over
// the active push transports. Stream subscribers receive a WS frame;
// when at least one WS subscriber is active for the event, we SKIP the
// HTTP webhook callback for that event so end-to-end tests can assert
// "WS replaced webhook for this install". Returns (advanced, delivered)
// where delivered counts the HTTP-callback deliveries (the WS broadcast
// count is reported by the hub via metrics).
func (s *server) evolveAndDeliver(ctx context.Context, n int) (advanced, delivered int) {
	logger := reqLogger(ctx, s.logger)
	results := s.store.EvolveSteps(n)
	advanced = len(results)
	if advanced == 0 {
		return 0, 0
	}
	for _, r := range results {
		eventName, resource := s.eventForResult(r)
		if eventName == "" {
			continue
		}
		ev := s.buildEvent(eventName, resource)

		// Stream-first: a connected WS subscriber for this event
		// short-circuits the HTTP callback path so we never deliver
		// both to the same install.
		if s.hub != nil {
			if n := s.hub.Broadcast(ctx, eventName, ev); n > 0 {
				logger.Debug("stream broadcast", "event", eventName, "ref", r.Reference, "subscribers", n)
				continue
			}
		}

		// "" and "none" both mean "no HMAC signing advertised" per
		// the contract — either way we don't drive the auto-emit
		// HTTP webhook path (the engine wouldn't verify).
		if s.cfg.webhookSignature == "" || s.cfg.webhookSignature == "none" {
			continue
		}
		callback, ok := s.store.findWebhookCallback(eventName)
		if !ok {
			logger.Debug("auto-emit skipped (no subscription)", "event", eventName, "ref", r.Reference)
			continue
		}
		if err := s.deliverWebhookEvent(ctx, callback, ev); err != nil {
			logger.Warn("auto-emit failed", "event", eventName, "ref", r.Reference, "error", err)
			continue
		}
		logger.Debug("auto-emit delivered", "event", eventName, "ref", r.Reference, "callback", callback)
		delivered++
	}
	return advanced, delivered
}

// buildEvent materialises a WebhookEvent envelope from the
// already-computed event name + resource. Used by both the stream
// broadcast and HTTP webhook paths so both transports ship byte-for-byte
// identical payloads.
func (s *server) buildEvent(eventName string, resource map[string]any) map[string]any {
	return map[string]any{
		"id":        "evt_" + time.Now().UTC().Format("20060102T150405.000000"),
		"type":      eventName,
		"createdAt": time.Now().UTC().Format(time.RFC3339),
		"resource":  resource,
	}
}

// eventForResult: EvolveResult → (event name, resource). ("", nil) for
// kinds not exposed as webhooks (orders/conversions — see
// contract/webhooks.md).
func (s *server) eventForResult(r EvolveResult) (string, map[string]any) {
	switch r.Kind {
	case "payment":
		p, ok := s.store.findPayment(r.Reference)
		if !ok {
			return "", nil
		}
		return "payment.updated", map[string]any{"payment": p}
	}
	// Order/conversion evolutions don't auto-emit: WebhookResponse has
	// no Order/Conversion field.
	return "", nil
}

// deliverWebhook signs and POSTs an event built ad-hoc from a resource
// (used by /_admin/trigger-webhook). Same envelope + HMAC as the
// auto-evolve path so VerifyWebhook+TranslateWebhook can't tell the
// difference.
func (s *server) deliverWebhook(ctx context.Context, callbackURL, eventName string, resource map[string]any) error {
	return s.deliverWebhookEvent(ctx, callbackURL, s.buildEvent(eventName, resource))
}

func (s *server) deliverWebhookEvent(ctx context.Context, callbackURL string, ev map[string]any) error {
	body, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Universal-Timestamp", timestamp)
	req.Header.Set("X-Universal-Signature", signHMAC(s.cfg.webhookSecret, timestamp, body))

	resp, err := webhookHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("callback returned %d", resp.StatusCode)
	}
	return nil
}

// parseListOpts extracts the shared pagination/filter query knobs. Bad
// values silently default (real PSPs don't surface parse errors either).
func parseListOpts(r *http.Request) listOpts {
	q := r.URL.Query()
	opts := listOpts{cursor: q.Get("cursor")}
	if v, err := strconv.Atoi(q.Get("page")); err == nil {
		opts.page = v
	}
	if v, err := strconv.Atoi(q.Get("pageSize")); err == nil {
		opts.pageSize = v
	}
	if raw := q.Get("updatedAtFrom"); raw != "" {
		if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			opts.updatedAtFrom = t
		} else if t, err := time.Parse(time.RFC3339, raw); err == nil {
			opts.updatedAtFrom = t
		}
	}
	return opts
}

func (s *server) handleCreatePayout(w http.ResponseWriter, r *http.Request) {
	var req initiationRequest
	if !readJSON(w, r, &req) {
		return
	}
	if err := validateInitiationRequest(req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	idem := r.Header.Get("Idempotency-Key")
	if idem == "" {
		writeErr(w, http.StatusBadRequest, "missing Idempotency-Key")
		return
	}
	resp := s.store.initiatePayout(idem, req)
	reqLogger(r.Context(), s.logger).Info("payout initiated",
		"reference", req.Reference,
		"amount", req.Amount,
		"asset", req.Asset,
		"mode", resp.Mode,
		"polling_id", resp.PollingID,
	)
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleGetPayout(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	resp := s.store.pollPayout(id)
	status := "(no payment yet)"
	if resp.Payment != nil {
		status = resp.Payment.Status
	}
	reqLogger(r.Context(), s.logger).Debug("payout polled", "polling_id", id, "payment_status", status)
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleReversePayout(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, reversalResponse("PAYOUT"))
}

func (s *server) handleCreateTransfer(w http.ResponseWriter, r *http.Request) {
	var req initiationRequest
	if !readJSON(w, r, &req) {
		return
	}
	if err := validateInitiationRequest(req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	idem := r.Header.Get("Idempotency-Key")
	if idem == "" {
		writeErr(w, http.StatusBadRequest, "missing Idempotency-Key")
		return
	}
	resp := s.store.initiateTransfer(idem, req)
	reqLogger(r.Context(), s.logger).Info("transfer initiated",
		"reference", req.Reference, "amount", req.Amount, "asset", req.Asset,
		"mode", resp.Mode, "polling_id", resp.PollingID,
	)
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleGetTransfer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	resp := s.store.pollTransfer(id)
	status := "(no payment yet)"
	if resp.Payment != nil {
		status = resp.Payment.Status
	}
	reqLogger(r.Context(), s.logger).Debug("transfer polled", "polling_id", id, "payment_status", status)
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleReverseTransfer(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, reversalResponse("TRANSFER"))
}

// reversalResponse synthesises a terminal REFUNDED payment for the
// reverse endpoints — engine only needs the REFUNDED status; the rest
// of the fields come from the engine-side PaymentInitiationReversal.
func reversalResponse(paymentType string) initiationResponse {
	now := time.Now().UTC()
	return initiationResponse{
		Mode: "terminal",
		Payment: &payment{
			Reference: "rev_" + now.Format("20060102T150405.000"),
			CreatedAt: now, UpdatedAt: now,
			Type: paymentType, Status: "REFUNDED",
		},
	}
}

func (s *server) handleCreateBankAccount(w http.ResponseWriter, r *http.Request) {
	var req bankAccountRequest
	if !readJSON(w, r, &req) {
		return
	}
	idem := r.Header.Get("Idempotency-Key")
	if idem == "" {
		writeErr(w, http.StatusBadRequest, "missing Idempotency-Key")
		return
	}
	logger := reqLogger(r.Context(), s.logger)
	asset := "EUR/2"
	if req.IBAN != nil && len(*req.IBAN) >= 2 {
		// IBAN country-code → asset is a fixture-only approximation.
		switch (*req.IBAN)[:2] {
		case "GB":
			asset = "GBP/2"
		case "US":
			asset = "USD/2"
		case "JP":
			asset = "JPY/0"
		}
	}
	// One critical section across the dedup-check, append, and
	// record so concurrent callers with the same Idempotency-Key
	// can't both pass the check and double-insert.
	created, dedup := s.store.createBankAccountLocked(idem, req.ID, req.Name, asset)
	if dedup {
		logger.Info("bank account dedup hit", "idem", idem, "reference", created.Reference)
	} else {
		logger.Info("bank account created", "reference", created.Reference, "asset", asset, "name", req.Name)
	}
	writeJSON(w, http.StatusOK, bankAccountResponse{RelatedAccount: created})
}

func (s *server) handleCreateWebhookSub(w http.ResponseWriter, r *http.Request) {
	var req webhookSubscriptionRequest
	if !readJSON(w, r, &req) {
		return
	}
	idem := r.Header.Get("Idempotency-Key")
	if idem == "" {
		writeErr(w, http.StatusBadRequest, "missing Idempotency-Key")
		return
	}
	logger := reqLogger(r.Context(), s.logger)
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	if existing, ok := s.store.idemWebhookSubID[idem]; ok {
		logger.Info("webhook subscription dedup hit", "idem", idem, "id", existing, "event", req.Name)
		writeJSON(w, http.StatusOK, webhookSubscriptionResponse{ID: existing, Name: req.Name})
		return
	}
	// Subscription IDs must be globally unique even when two installs
	// register the same event — keyed only on `name` we'd silently
	// overwrite the previous callback. A monotonic counter under the
	// store lock gives uniqueness with no extra deps.
	s.store.webhookSubSeq++
	id := fmt.Sprintf("sub_%s_%d", req.Name, s.store.webhookSubSeq)
	s.store.idemWebhookSubID[idem] = id
	s.store.webhookSubs[id] = webhookSub{ID: id, Name: req.Name, CallbackURL: req.CallbackURL}
	logger.Info("webhook subscription created",
		"id", id, "event", req.Name, "callback", req.CallbackURL,
	)
	writeJSON(w, http.StatusOK, webhookSubscriptionResponse{ID: id, Name: req.Name})
}

func (s *server) handleDeleteWebhookSub(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.store.mu.Lock()
	_, existed := s.store.webhookSubs[id]
	delete(s.store.webhookSubs, id)
	s.store.mu.Unlock()
	reqLogger(r.Context(), s.logger).Info("webhook subscription deleted", "id", id, "existed", existed)
	w.WriteHeader(http.StatusNoContent)
}

// handleAdminTrigger pushes one signed seed-backed event so the
// VerifyWebhook → TranslateWebhook → engine-store loop runs end-to-end.
// Out-of-contract.
func (s *server) handleAdminTrigger(w http.ResponseWriter, r *http.Request) {
	logger := reqLogger(r.Context(), s.logger)
	name := r.URL.Query().Get("name")
	if name == "" {
		writeErr(w, http.StatusBadRequest, "missing ?name=")
		return
	}
	url, ok := s.store.findWebhookCallback(name)
	if !ok {
		logger.Warn("admin trigger failed: no subscription", "event", name)
		writeErr(w, http.StatusNotFound, "no subscription for "+name)
		return
	}

	resource, err := s.materialiseEventResource(name)
	if err != nil {
		logger.Warn("admin trigger failed: cannot materialise resource", "event", name, "error", err)
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.deliverWebhook(r.Context(), url, name, resource); err != nil {
		logger.Warn("admin trigger delivery failed", "event", name, "callback", url, "error", err)
		writeErr(w, http.StatusBadGateway, "delivery failed: "+err.Error())
		return
	}
	logger.Info("admin trigger delivered", "event", name, "callback", url)
	w.WriteHeader(http.StatusOK)
}

// materialiseEventResource builds the resource payload for an event
// per contract/universal-events.md, sourced from seed data.
func (s *server) materialiseEventResource(name string) (map[string]any, error) {
	switch name {
	case "account.created", "account.updated":
		acc, ok := s.store.firstAccount()
		if !ok {
			return nil, errMissingSeed("account")
		}
		return map[string]any{"account": acc}, nil
	case "external_account.created":
		s.store.mu.RLock()
		defer s.store.mu.RUnlock()
		if len(s.store.externalAccounts) == 0 {
			return nil, errMissingSeed("externalAccount")
		}
		return map[string]any{"externalAccount": s.store.externalAccounts[0]}, nil
	case "balance.updated":
		acc, ok := s.store.firstAccount()
		if !ok {
			return nil, errMissingSeed("account")
		}
		bs := s.store.accountBalances(acc.Reference)
		if len(bs) == 0 {
			return nil, errMissingSeed("balance")
		}
		return map[string]any{"balance": bs[0]}, nil
	case "payment.created", "payment.updated":
		p, ok := s.store.firstPayment()
		if !ok {
			return nil, errMissingSeed("payment")
		}
		return map[string]any{"payment": p}, nil
	case "payment.deleted":
		ref := s.store.firstPaymentRef()
		if ref == "" {
			return nil, errMissingSeed("payment")
		}
		return map[string]any{"paymentToDelete": ref}, nil
	case "payment.cancelled":
		ref := s.store.firstPaymentRef()
		if ref == "" {
			return nil, errMissingSeed("payment")
		}
		return map[string]any{"paymentToCancel": ref}, nil
	default:
		return nil, fmt.Errorf("unknown webhook event %q (note: order.* and conversion.* are not webhook-able — orders/conversions are pull-only)", name)
	}
}

func errMissingSeed(kind string) error { return fmt.Errorf("no seeded %s to emit", kind) }

// handleAdminEvolve advances N records (default 1) and reports both
// the advanced count and the webhook auto-emissions.
func (s *server) handleAdminEvolve(w http.ResponseWriter, r *http.Request) {
	n := 1
	if raw := r.URL.Query().Get("n"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			n = v
		}
	}
	advanced, delivered := s.evolveAndDeliver(r.Context(), n)
	reqLogger(r.Context(), s.logger).Info("admin evolve",
		"requested", n, "advanced", advanced, "webhooks_delivered", delivered,
	)
	writeJSON(w, http.StatusOK, map[string]int{"advanced": advanced, "webhooksDelivered": delivered})
}

// validateInitiationRequest enforces minimal sanity on incoming
// payout/transfer bodies so the store never sees a malformed amount
// (which mustParseInt silently maps to 0 — a deceptively "valid"
// terminal small-amount payment in the test fixture).
func validateInitiationRequest(req initiationRequest) error {
	if req.Reference == "" {
		return fmt.Errorf("reference is required")
	}
	if _, err := strconv.ParseInt(req.Amount, 10, 64); err != nil {
		return fmt.Errorf("amount %q is not a decimal integer", req.Amount)
	}
	if req.Asset == "" {
		return fmt.Errorf("asset is required")
	}
	if req.SourceAccountReference == "" || req.DestinationAccountReference == "" {
		return fmt.Errorf("source and destination account references are required")
	}
	return nil
}

func signHMAC(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Encode failure here means the connection died mid-write; logging
	// every dropped connection isn't actionable.
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, errorResponse{Title: http.StatusText(status), Status: status, Detail: detail})
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	return true
}
