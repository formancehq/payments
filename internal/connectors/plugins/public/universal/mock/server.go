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

// server is the HTTP front for the mock counterparty. It wires every
// contract endpoint to the in-memory store, plus a single bearer-auth
// middleware applied to every route except the trivial /healthz.
type server struct {
	cfg    mockConfig
	store  *store
	mux    *http.ServeMux
	logger *slog.Logger
}

// newServer wires routes and returns the server itself. Use Handler()
// to obtain the auth-wrapped http.Handler ready to plug into
// http.Server. Tests in the same package construct via newServer to
// reach internal helpers (evolveAndDeliver, etc.) directly.
func newServer(cfg mockConfig, st *store, logger *slog.Logger) *server {
	if logger == nil {
		logger = newLogger("info")
	}
	s := &server{cfg: cfg, store: st, mux: http.NewServeMux(), logger: logger}
	s.routes()
	return s
}

// Handler returns the bearer-auth-wrapped + request-logged http.Handler.
// Layering, outside-in: logging → auth → mux. So unauthenticated
// requests still produce a "← response" log line.
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

	// Debug endpoints (NOT in the contract):
	//
	//   POST /_admin/trigger-webhook?name=<event>
	//     Pushes one synthetic, signed event of the given type to the
	//     registered callback URL so the VerifyWebhook + TranslateWebhook
	//     code path can be exercised end-to-end without waiting for the
	//     mock to autonomously emit.
	//
	//   POST /_admin/evolve?n=<int>
	//     Advances up to N non-terminal payments / orders one step
	//     through their state machine, bumping `updatedAt`. Drives the
	//     engine's PaymentAdjustment / OrderAdjustment derivation
	//     end-to-end on subsequent FetchNext* polls.
	s.mux.HandleFunc("POST /_admin/trigger-webhook", s.handleAdminTrigger)
	s.mux.HandleFunc("POST /_admin/evolve", s.handleAdminEvolve)
}

// requireAuth is a tiny bearer-auth middleware. We never log the token
// itself; we log the auth outcome so misconfigurations are obvious.
func (s *server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
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

// ---- handlers --------------------------------------------------------------

func (s *server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	// Surface the install discovery as a high-signal log line; this
	// is THE event that signals "someone is installing the connector".
	reqLogger(r.Context(), s.logger).Info("capabilities discovery",
		"supported_count", len(s.cfg.capabilities),
		"signature_scheme", s.cfg.webhookSignature,
	)
	writeJSON(w, http.StatusOK, capabilities{
		Supported: s.cfg.capabilities,
		Features: features{
			Pagination:       "cursor",
			WebhookSignature: s.cfg.webhookSignature,
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
// evolution is enabled AND the engine has not registered any webhook
// subscriptions. The gating matters for two reasons:
//
//  1. With webhooks active, the engine relies on pushed events for
//     low-latency state propagation; polls become slow heartbeats and
//     should not be the source of state mutations.
//
//  2. With no webhooks, polls are the engine's only signal — so each
//     poll evolves the dataset, which means each poll cycle the engine
//     observes a new batch of adjustments to derive.
//
// Evolution happens BEFORE serving the response so the engine sees the
// new state on the same poll. With the engine's 20-minute minimum
// polling period, this matches how a real PSP would surface a steady
// drip of status changes between polls.
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

// evolveAndDeliver advances `n` records and, for every transition, pushes
// a matching webhook event (e.g. "payment.updated") to the registered
// callback if a subscription exists. This is the auto-emit path that
// closes the loop in webhook-mode: any state change driven by
// /_admin/evolve or the background ticker reaches the engine
// immediately rather than waiting for the next poll.
//
// Returns the number of records advanced (== number of evolution
// results) and, separately, the number of webhooks that were actually
// dispatched (≤ advanced; the count is lower when there's no
// subscription for the corresponding event type).
func (s *server) evolveAndDeliver(ctx context.Context, n int) (advanced, delivered int) {
	logger := reqLogger(ctx, s.logger)
	results := s.store.EvolveSteps(n)
	advanced = len(results)
	if s.cfg.webhookSignature == "" || advanced == 0 {
		return advanced, 0
	}
	for _, r := range results {
		eventName, resource := s.eventForResult(r)
		if eventName == "" {
			continue
		}
		callback, ok := s.store.findWebhookCallback(eventName)
		if !ok {
			logger.Debug("auto-emit skipped (no subscription)", "event", eventName, "ref", r.Reference)
			continue
		}
		if err := s.deliverWebhook(ctx, callback, eventName, resource); err != nil {
			logger.Warn("auto-emit failed", "event", eventName, "ref", r.Reference, "error", err)
			continue
		}
		logger.Debug("auto-emit delivered", "event", eventName, "ref", r.Reference, "callback", callback)
		delivered++
	}
	return advanced, delivered
}

// eventForResult turns an EvolveResult into the event name + typed
// resource payload the engine's TranslateWebhook expects. Returns
// ("", nil) for kinds the universal contract doesn't expose as a
// webhook event (orders, conversions — see contract/webhooks.md
// "Subscribed events" for why), or when the post-mutation lookup
// fails (record was somehow removed between evolve and dispatch —
// shouldn't happen, but a safety net).
func (s *server) eventForResult(r EvolveResult) (string, map[string]any) {
	switch r.Kind {
	case "payment":
		p, ok := s.store.findPayment(r.Reference)
		if !ok {
			return "", nil
		}
		return "payment.updated", map[string]any{"payment": p}
	}
	// Order / conversion evolutions intentionally don't auto-emit:
	// the engine's WebhookResponse has no Order or Conversion field,
	// so subscribing to them at install would be misleading.
	return "", nil
}

// deliverWebhook signs and POSTs one event to the given callback URL.
// Reuses the same envelope + HMAC scheme as /_admin/trigger-webhook so
// the engine's VerifyWebhook + TranslateWebhook code path doesn't see a
// distinction between manual triggers and auto-emitted events.
func (s *server) deliverWebhook(ctx context.Context, callbackURL, eventName string, resource map[string]any) error {
	body, err := json.Marshal(map[string]any{
		"id":        "evt_" + time.Now().UTC().Format("20060102T150405.000000"),
		"type":      eventName,
		"createdAt": time.Now().UTC().Format(time.RFC3339),
		"resource":  resource,
	})
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("callback returned %d", resp.StatusCode)
	}
	return nil
}

// parseListOpts extracts the cursor/page/pageSize/updatedAtFrom query
// parameters every paginated GET shares. Bad values silently default —
// counterparties wouldn't surface query-string parse errors either, and
// the engine treats an empty page as "no more rows".
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
// reverse-payout / reverse-transfer endpoints. Callers don't need the
// request body to be inspected because the engine only cares that a
// REFUNDED PSPPayment came back; its amount/asset/refs come from the
// engine-side PaymentInitiationReversal aggregate.
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
	if existing, ok := s.store.resolveBankAccount(idem); ok {
		logger.Info("bank account dedup hit", "idem", idem, "reference", existing)
		writeJSON(w, http.StatusOK, bankAccountResponse{
			RelatedAccount: account{Reference: existing, CreatedAt: time.Now().UTC(), Name: &req.Name},
		})
		return
	}
	ref := "acct_ext_ba_" + req.ID
	asset := "EUR/2"
	if req.IBAN != nil && len(*req.IBAN) >= 2 {
		// Cheap mapping: country code → asset. Good enough for fixtures.
		switch (*req.IBAN)[:2] {
		case "GB":
			asset = "GBP/2"
		case "US":
			asset = "USD/2"
		case "JP":
			asset = "JPY/0"
		}
	}
	created := account{
		Reference: ref, CreatedAt: time.Now().UTC(),
		Name: &req.Name, DefaultAsset: &asset,
	}
	s.store.addExternalAccount(created)
	s.store.recordBankAccount(idem, ref)
	logger.Info("bank account created", "reference", ref, "asset", asset, "name", req.Name)
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
	id := "sub_" + req.Name
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

// handleAdminTrigger pushes one signed event to the registered callback URL
// for the given subscription name. The payload is materialised from seed
// data so the engine's TranslateWebhook code path actually has a typed
// resource to convert (Payment, Account, ...) — verifying the full
// VerifyWebhook → TranslateWebhook → engine-store loop end-to-end.
//
// Explicitly not part of the contract; lives behind /_admin/.
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

// materialiseEventResource builds a `resource` payload that matches the
// shape contract/universal-events.md declares for the given event name —
// using seed data so the engine sees a realistic record. Unknown event
// names fail loudly so a debug typo doesn't silently push a malformed
// event the engine would then reject.
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

// handleAdminEvolve advances N non-terminal payments / orders through
// their state machines so the engine's adjustment derivation can be
// observed end-to-end. Defaults to n=1; reports the count actually
// advanced (may be lower if everything is already terminal) and the
// number of webhook events auto-emitted to registered subscriptions.
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

// ---- helpers ---------------------------------------------------------------

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
	// json encode failure on a writer at this stage means the connection
	// died mid-write; nothing useful we can do, slog noise on every
	// dropped connection isn't actionable. Discard.
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
