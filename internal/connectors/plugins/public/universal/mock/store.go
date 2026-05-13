package main

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

// store is a single-tenant, in-memory data layer. Everything is keyed by
// the entity's primary reference. A single sync.RWMutex protects all maps —
// fine for the fixture's purpose; multi-tenancy will replace this with a
// per-tenant struct.
//
// Adjustment evolution is driven by per-record "lanes" — pre-planned
// sequences of next statuses. EvolveSteps pops one entry off the next
// non-empty lane on every call, bumps updatedAt, and the engine sees a
// fresh PaymentAdjustment / OrderAdjustment on the following poll.
type store struct {
	mu               sync.RWMutex
	cfg              mockConfig
	logger           *slog.Logger
	accounts         []account
	externalAccounts []account
	balances         map[string][]balance // accountReference -> balances
	payments         []payment
	orders           []order
	conversions      []conversion
	others           map[string][]other

	// Per-record evolution lanes. Each entry is a queue of next statuses
	// (and, for orders, fill-quantity bumps) that get popped on every
	// EvolveSteps call. Empty queue ⇒ record is terminal for evolution
	// purposes. Round-robin cursors below ensure progress spreads
	// evenly across the dataset rather than draining one record before
	// touching the next.
	paymentLanes    map[string][]string
	orderLanes      map[string][]orderStep
	conversionLanes map[string][]string

	evolvePaymentCursor    int
	evolveOrderCursor      int
	evolveConversionCursor int

	// Idempotency dedups POST requests by Idempotency-Key (one map per
	// resource family is overkill but keeps the store readable).
	idemPayout       map[string]string // key -> payout ID (or terminal sentinel)
	idemTransfer     map[string]string
	idemBankAccount  map[string]string
	idemWebhookSubID map[string]string

	pollingPayouts   map[string]*pollEntry // pollingID -> entry
	pollingTransfers map[string]*pollEntry
	webhookSubs      map[string]webhookSub // sub.ID -> sub
}

// orderStep is one entry in an order's evolution lane: the status the
// order should transition to and the resulting fill (as a percentage of
// `BaseQuantityOrdered`, 0–100). The percentage representation lets us
// keep the lane definition human-readable while still emitting concrete
// minor-unit fill quantities the engine can compare for adjustment
// dedup purposes.
type orderStep struct {
	status  string
	fillPct int
}

// Seed sizes are tuned so the engine's default PAGE_SIZE (100) needs at
// least two pages on the largest endpoints — exercising the cursor-based
// pagination state machine end-to-end on the plugin side.
const (
	seedInternalAccounts = 5
	seedExternalAccounts = 5
	seedPayments         = 250
	seedOrders           = 150
	seedConversions      = 50
	seedOthers           = 30
)

func newStore(cfg mockConfig, logger *slog.Logger) *store {
	if logger == nil {
		logger = newLogger("info")
	}
	s := &store{
		cfg:              cfg,
		logger:           logger,
		balances:         map[string][]balance{},
		others:           map[string][]other{},
		paymentLanes:     map[string][]string{},
		orderLanes:       map[string][]orderStep{},
		conversionLanes:  map[string][]string{},
		idemPayout:       map[string]string{},
		idemTransfer:     map[string]string{},
		idemBankAccount:  map[string]string{},
		idemWebhookSubID: map[string]string{},
		pollingPayouts:   map[string]*pollEntry{},
		pollingTransfers: map[string]*pollEntry{},
		webhookSubs:      map[string]webhookSub{},
	}
	s.seed()
	logger.Info("store seeded",
		"accounts", len(s.accounts),
		"external_accounts", len(s.externalAccounts),
		"payments", len(s.payments),
		"orders", len(s.orders),
		"conversions", len(s.conversions),
		"others", len(s.others["report"]),
	)
	return s
}

// paymentLaneTemplates is the catalogue of trajectories every seeded
// payment can follow. Each lane is a complete adjustment story —
// distributing the catalogue across the 250 seeded payments guarantees
// every PaymentStatus transition the engine's
// FromPaymentDataToPaymentInitiationAdjustment table can produce gets
// exercised somewhere in the dataset.
var paymentLaneTemplates = [][]string{
	// Card-style auth/capture flows
	{"AUTHORISATION", "CAPTURE", "SUCCEEDED"},
	{"AUTHORISATION", "CAPTURE_FAILED"},
	// Direct success / failure
	{"SUCCEEDED"},
	{"FAILED"},
	{"CANCELLED"},
	{"EXPIRED"},
	// Refunds
	{"SUCCEEDED", "REFUNDED"},
	{"SUCCEEDED", "REFUNDED", "REFUND_REVERSED"},
	{"SUCCEEDED", "REFUNDED_FAILURE"},
	// Disputes
	{"SUCCEEDED", "DISPUTE", "DISPUTE_WON"},
	{"SUCCEEDED", "DISPUTE", "DISPUTE_LOST"},
}

// orderLaneTemplates covers every OrderStatus terminal + the partial-fill
// progression. PARTIALLY_FILLED appears more than once in the FILL trajectory
// so the engine's per-fill OrderAdjustment dedup (which keys on
// BaseQuantityFilled) is exercised even when status doesn't change.
var orderLaneTemplates = [][]orderStep{
	{{"OPEN", 0}, {"PARTIALLY_FILLED", 25}, {"PARTIALLY_FILLED", 50}, {"PARTIALLY_FILLED", 75}, {"FILLED", 100}},
	{{"OPEN", 0}, {"FILLED", 100}},
	{{"OPEN", 0}, {"PARTIALLY_FILLED", 33}, {"CANCELLED", 33}},
	{{"OPEN", 0}, {"EXPIRED", 0}},
	{{"FAILED", 0}},
}

// conversionLaneTemplates: simple single-step terminals — conversions are
// latest-wins in the engine and don't carry an adjustment array.
var conversionLaneTemplates = [][]string{
	{"COMPLETED"},
	{"FAILED"},
}

// seed populates the store with deterministic but realistic-looking
// fixtures. Timestamps are monotonic per record (CreatedAt and UpdatedAt
// strictly increasing across the seed slice) so `updatedAtFrom` cursoring
// on the engine side behaves as it would against a real PSP.
func (s *store) seed() {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tick := func(i int) time.Time { return base.Add(time.Duration(i) * time.Minute) }

	assets := []string{"EUR/2", "USD/2", "GBP/2", "JPY/0"}
	for i := 0; i < seedInternalAccounts; i++ {
		ref := fmt.Sprintf("acct_internal_%03d", i)
		asset := assets[i%len(assets)]
		s.accounts = append(s.accounts, account{
			Reference:    ref,
			CreatedAt:    tick(i),
			Name:         ptr(fmt.Sprintf("Operating %s #%d", asset, i)),
			DefaultAsset: ptr(asset),
		})
		s.balances[ref] = []balance{{
			AccountReference: ref,
			CreatedAt:        tick(i),
			Amount:           strconv.FormatInt(1_000_000+int64(i)*10_000, 10),
			Asset:            asset,
		}}
	}
	for i := 0; i < seedExternalAccounts; i++ {
		ref := fmt.Sprintf("acct_ext_%03d", i)
		s.externalAccounts = append(s.externalAccounts, account{
			Reference:    ref,
			CreatedAt:    tick(i),
			Name:         ptr(fmt.Sprintf("Counterparty External #%d", i)),
			DefaultAsset: ptr(assets[i%len(assets)]),
		})
	}

	// Every payment, order and conversion starts in PENDING and is
	// assigned a lane chosen round-robin from the catalogue. EvolveSteps
	// pops one entry off the lane per call so the engine sees a fresh
	// adjustment each time, and the dataset eventually reaches every
	// terminal status in the engine's enum.
	types := []string{"PAYIN", "PAYOUT", "TRANSFER"}
	for i := 0; i < seedPayments; i++ {
		t := tick(i)
		srcRef := s.accounts[i%len(s.accounts)].Reference
		dstRef := s.externalAccounts[i%len(s.externalAccounts)].Reference
		ref := fmt.Sprintf("pay_%05d", i)
		s.payments = append(s.payments, payment{
			Reference:                   ref,
			CreatedAt:                   t,
			UpdatedAt:                   t,
			Type:                        types[i%len(types)],
			Status:                      "PENDING",
			Amount:                      strconv.Itoa(100 + i*10),
			Asset:                       assets[i%len(assets)],
			SourceAccountReference:      &srcRef,
			DestinationAccountReference: &dstRef,
		})
		s.paymentLanes[ref] = append([]string(nil), paymentLaneTemplates[i%len(paymentLaneTemplates)]...)
	}

	directions := []string{"BUY", "SELL"}
	for i := 0; i < seedOrders; i++ {
		t := tick(i)
		ref := s.accounts[i%len(s.accounts)].Reference
		ordered := int64(1_000_000 + i*100)
		oref := fmt.Sprintf("ord_%05d", i)
		s.orders = append(s.orders, order{
			Reference:                   oref,
			CreatedAt:                   t,
			UpdatedAt:                   t,
			Direction:                   directions[i%len(directions)],
			Type:                        "MARKET",
			Status:                      "PENDING",
			// MARKET orders are canonically immediate-or-cancel.
			// The engine's storage column is `time_in_force NOT NULL`
			// so we MUST emit a real value — `TIME_IN_FORCE_UNKNOWN`
			// (the default for an empty wire field) is rejected with a
			// SQL error by the engine's bun layer.
			TimeInForce:                 "IOC",
			SourceAsset:                 "EUR/2",
			DestinationAsset:            "BTC/8",
			BaseQuantityOrdered:         strconv.FormatInt(ordered, 10),
			BaseQuantityFilled:          "0",
			QuoteAmount:                 strconv.FormatInt(ordered*85/10000, 10),
			QuoteAsset:                  "EUR/2",
			SourceAccountReference:      &ref,
			DestinationAccountReference: &ref,
		})
		s.orderLanes[oref] = append([]orderStep(nil), orderLaneTemplates[i%len(orderLaneTemplates)]...)
	}

	for i := 0; i < seedConversions; i++ {
		t := tick(i)
		ref := s.accounts[i%len(s.accounts)].Reference
		amt := int64(1_000_000 + i*100)
		cref := fmt.Sprintf("conv_%05d", i)
		s.conversions = append(s.conversions, conversion{
			Reference:                   cref,
			CreatedAt:                   t,
			Status:                      "PENDING",
			SourceAsset:                 "USDC/6",
			DestinationAsset:            "USD/2",
			SourceAmount:                strconv.FormatInt(amt, 10),
			DestinationAmount:           strconv.FormatInt(amt/100, 10),
			SourceAccountReference:      &ref,
			DestinationAccountReference: &ref,
		})
		s.conversionLanes[cref] = append([]string(nil), conversionLaneTemplates[i%len(conversionLaneTemplates)]...)
	}

	for i := 0; i < seedOthers; i++ {
		s.others["report"] = append(s.others["report"], other{
			ID:   fmt.Sprintf("rep_%05d", i),
			Data: map[string]any{"index": i, "kind": "monthly"},
		})
	}
}

func ptr[T any](v T) *T { return &v }

// EvolveResult identifies a single advance: which kind of record
// changed and its primary reference. Returned by EvolveSteps so the
// caller (typically the server) can push a matching webhook event for
// every transition when a subscription exists.
type EvolveResult struct {
	Kind      string // "payment" | "order" | "conversion"
	Reference string
}

// EvolveSteps advances up to `n` records by popping one entry off each
// non-empty payment / order / conversion lane, bumping updatedAt, and
// returning the per-record results. It rotates across primitive types
// so a single Evolve(N) call exercises payment + order + conversion
// state machines in parallel rather than draining one before touching
// the next.
//
// Each advance gets a strictly-monotonic timestamp: a shared `now` cursor
// is incremented by one microsecond per record. Without this, the engine's
// `updatedAtFrom` high-water cursor would skip records that share an
// identical UpdatedAt — effectively losing adjustments.
//
// Driven by:
//   - The background ticker spawned in main() when MOCK_AUTO_EVOLVE_INTERVAL > 0.
//   - The explicit POST /_admin/evolve?n=K endpoint (tests + manual control).
//   - Each paginated GET when evolveOnPoll is enabled and no webhook
//     subscriptions are registered.
//
// Returns the slice of advanced records (length may be lower than n if
// every lane is exhausted). Safe to call concurrently.
func (s *store) EvolveSteps(n int) []EvolveResult {
	if n <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	cur := newTimeCursor(time.Now().UTC())
	results := make([]EvolveResult, 0, n)
	for len(results) < n {
		stepped := false
		if ref, ok := s.evolveOnePaymentLocked(cur.next()); ok {
			results = append(results, EvolveResult{Kind: "payment", Reference: ref})
			stepped = true
		}
		if len(results) >= n {
			break
		}
		if ref, ok := s.evolveOneOrderLocked(cur.next()); ok {
			results = append(results, EvolveResult{Kind: "order", Reference: ref})
			stepped = true
		}
		if len(results) >= n {
			break
		}
		if ref, ok := s.evolveOneConversionLocked(cur.next()); ok {
			results = append(results, EvolveResult{Kind: "conversion", Reference: ref})
			stepped = true
		}
		if !stepped {
			break // every lane drained
		}
	}
	return results
}

// timeCursor produces a strictly-monotonic series of timestamps starting
// at `t`, each one microsecond apart. Lets one EvolveSteps call stamp
// every advanced record with a distinct UpdatedAt so the engine's
// high-water cursor never skips an adjustment.
type timeCursor struct{ t time.Time }

func newTimeCursor(t time.Time) *timeCursor { return &timeCursor{t: t} }

func (c *timeCursor) next() time.Time {
	c.t = c.t.Add(time.Microsecond)
	return c.t
}

// evolveOnePaymentLocked pops one status off the next non-empty payment
// lane and applies it. Cursor-driven round-robin so successive calls
// touch different records — auto-evolve produces uniform progress
// instead of fully draining record 0 before touching record 1. Returns
// the affected payment's reference + true; ("", false) when every lane
// is drained.
func (s *store) evolveOnePaymentLocked(now time.Time) (string, bool) {
	for tries := 0; tries < len(s.payments); tries++ {
		i := (s.evolvePaymentCursor + tries) % len(s.payments)
		ref := s.payments[i].Reference
		lane := s.paymentLanes[ref]
		if len(lane) == 0 {
			continue
		}
		from := s.payments[i].Status
		s.payments[i].Status = lane[0]
		s.payments[i].UpdatedAt = now
		s.paymentLanes[ref] = lane[1:]
		s.evolvePaymentCursor = i + 1
		s.logger.Debug("evolved payment", "ref", ref, "from", from, "to", lane[0], "remaining", len(lane)-1)
		return ref, true
	}
	return "", false
}

// evolveOneOrderLocked pops one orderStep off the next non-empty order
// lane (cursor-driven round-robin). Fill quantity is computed as
// percent-of-ordered so the engine's adjustment dedup (which keys on
// BaseQuantityFilled) treats two PARTIALLY_FILLED entries with different
// fills as distinct adjustments.
func (s *store) evolveOneOrderLocked(now time.Time) (string, bool) {
	for tries := 0; tries < len(s.orders); tries++ {
		i := (s.evolveOrderCursor + tries) % len(s.orders)
		ref := s.orders[i].Reference
		lane := s.orderLanes[ref]
		if len(lane) == 0 {
			continue
		}
		step := lane[0]
		from := s.orders[i].Status
		s.orders[i].Status = step.status
		ordered := mustParseInt(s.orders[i].BaseQuantityOrdered)
		s.orders[i].BaseQuantityFilled = strconv.FormatInt(ordered*int64(step.fillPct)/100, 10)
		s.orders[i].UpdatedAt = now
		s.orderLanes[ref] = lane[1:]
		s.evolveOrderCursor = i + 1
		s.logger.Debug("evolved order", "ref", ref, "from", from, "to", step.status, "fill_pct", step.fillPct, "remaining", len(lane)-1)
		return ref, true
	}
	return "", false
}

// evolveOneConversionLocked pops one status off the next non-empty
// conversion lane (cursor-driven). Conversions don't carry adjustment
// history in the engine, but evolving them still validates the
// FetchNextConversions path observes the new status.
func (s *store) evolveOneConversionLocked(now time.Time) (string, bool) {
	for tries := 0; tries < len(s.conversions); tries++ {
		i := (s.evolveConversionCursor + tries) % len(s.conversions)
		ref := s.conversions[i].Reference
		lane := s.conversionLanes[ref]
		if len(lane) == 0 {
			continue
		}
		from := s.conversions[i].Status
		s.conversions[i].Status = lane[0]
		s.conversions[i].CreatedAt = now // conversions only carry CreatedAt as freshness anchor
		s.conversionLanes[ref] = lane[1:]
		s.evolveConversionCursor = i + 1
		s.logger.Debug("evolved conversion", "ref", ref, "from", from, "to", lane[0], "remaining", len(lane)-1)
		return ref, true
	}
	return "", false
}

// pollEntry is the polling state machine the mock uses to demonstrate the
// terminal-or-polling pattern. We start in PENDING; after `transitionsLeft`
// polls we flip to SUCCEEDED. This is enough to exercise the
// PluginPollPayoutStatus Temporal workflow end-to-end.
type pollEntry struct {
	id              string
	reference       string
	asset           string
	amount          string
	srcAccount      string
	dstAccount      string
	transitionsLeft int
	terminalStatus  string
	createdAt       time.Time
}

type webhookSub struct {
	ID          string
	Name        string
	CallbackURL string
}

// --- pagination helpers ------------------------------------------------------

// paginate slices `items` into a single page using either the opaque
// cursor (a base64-encoded integer offset, easy to debug) or a 1-based
// page number. Returns the page contents, the cursor for the next page
// (empty if exhausted), and whether more rows remain. pageSize 0 falls
// back to 100 to match the engine's default PAGE_SIZE.
func paginate[T any](items []T, cursor string, page, pageSize int) ([]T, string, bool) {
	if pageSize <= 0 {
		pageSize = 100
	}
	offset := 0
	if cursor != "" {
		if n, err := decodeCursor(cursor); err == nil {
			offset = n
		}
	} else if page > 1 {
		offset = (page - 1) * pageSize
	}

	if offset >= len(items) {
		return nil, "", false
	}
	end := offset + pageSize
	if end > len(items) {
		end = len(items)
	}
	hasMore := end < len(items)
	next := ""
	if hasMore {
		next = encodeCursor(end)
	}
	return items[offset:end], next, hasMore
}

func encodeCursor(offset int) string {
	return base64.URLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func decodeCursor(s string) (int, error) {
	raw, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(raw))
}

// listOpts bundles the query parameters every paginated GET shares.
type listOpts struct {
	cursor        string
	page          int
	pageSize      int
	updatedAtFrom time.Time
}

// --- paginated read helpers --------------------------------------------------

func (s *store) listAccounts(opts listOpts) ([]account, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return paginate(filterAccountsByUpdatedAt(s.accounts, opts.updatedAtFrom), opts.cursor, opts.page, opts.pageSize)
}

func (s *store) listExternalAccounts(opts listOpts) ([]account, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return paginate(filterAccountsByUpdatedAt(s.externalAccounts, opts.updatedAtFrom), opts.cursor, opts.page, opts.pageSize)
}

func (s *store) accountBalances(accountID string) []balance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]balance(nil), s.balances[accountID]...)
}

func (s *store) listPayments(opts listOpts) ([]payment, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	filtered := s.payments
	if !opts.updatedAtFrom.IsZero() {
		filtered = filtered[:0:0]
		for _, p := range s.payments {
			if p.UpdatedAt.After(opts.updatedAtFrom) {
				filtered = append(filtered, p)
			}
		}
	}
	return paginate(filtered, opts.cursor, opts.page, opts.pageSize)
}

func (s *store) listOrders(opts listOpts) ([]order, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	filtered := s.orders
	if !opts.updatedAtFrom.IsZero() {
		filtered = filtered[:0:0]
		for _, o := range s.orders {
			if o.UpdatedAt.After(opts.updatedAtFrom) {
				filtered = append(filtered, o)
			}
		}
	}
	return paginate(filtered, opts.cursor, opts.page, opts.pageSize)
}

func (s *store) listConversions(opts listOpts) ([]conversion, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return paginate(s.conversions, opts.cursor, opts.page, opts.pageSize)
}

func (s *store) listOthers(name string, opts listOpts) ([]other, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return paginate(s.others[name], opts.cursor, opts.page, opts.pageSize)
}

// filterAccountsByUpdatedAt approximates an "updated since" filter for
// accounts (which only carry CreatedAt on the wire). The contract treats
// CreatedAt as the freshness anchor for accounts since they are largely
// append-only.
func filterAccountsByUpdatedAt(in []account, since time.Time) []account {
	if since.IsZero() {
		return in
	}
	out := make([]account, 0, len(in))
	for _, a := range in {
		if a.CreatedAt.After(since) {
			out = append(out, a)
		}
	}
	return out
}

// --- payout mutate -----------------------------------------------------------

// initiatePayout returns either a terminal payment (≤ €100) or a polling ID
// that progresses to SUCCEEDED after a few polls (> €100). This single rule
// covers both branches of the contract's terminal-or-polling envelope and
// gives Temporal something to actually schedule.
//
// The synthesized Payment is also appended to s.payments so a subsequent
// /v1/payments poll surfaces it — exactly how a real PSP behaves: a payout
// is also a transaction. The Payment.Reference matches the engine-side
// initiation reference so the engine can correlate the
// PaymentInitiationAdjustment trail with the PaymentAdjustment trail.
func (s *store) initiatePayout(idemKey string, req initiationRequest) initiationResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.initiateLocked(idemKey, req, "PAYOUT", s.idemPayout, s.pollingPayouts, "ppayout_")
}

func (s *store) initiateTransferLocked(idemKey string, req initiationRequest) initiationResponse {
	return s.initiateLocked(idemKey, req, "TRANSFER", s.idemTransfer, s.pollingTransfers, "ptransfer_")
}

// initiateLocked is the shared implementation used by both payouts and
// transfers. The caller MUST hold s.mu.Lock().
func (s *store) initiateLocked(
	idemKey string,
	req initiationRequest,
	paymentType string,
	idem map[string]string,
	polling map[string]*pollEntry,
	pollingPrefix string,
) initiationResponse {
	if existing, ok := idem[idemKey]; ok {
		if entry := polling[existing]; entry == nil {
			return initiationResponse{Mode: "terminal", Payment: terminalPayment(req, paymentType)}
		}
		return initiationResponse{Mode: "polling", PollingID: existing}
	}

	if mustParseInt(req.Amount) <= 10000 { // ≤ €100.00 minor units → terminal
		idem[idemKey] = req.Reference
		pay := terminalPayment(req, paymentType)
		s.upsertPaymentLocked(pay)
		return initiationResponse{Mode: "terminal", Payment: pay}
	}

	id := pollingPrefix + req.Reference
	idem[idemKey] = id
	polling[id] = &pollEntry{
		id: id, reference: req.Reference, asset: req.Asset, amount: req.Amount,
		srcAccount: req.SourceAccountReference, dstAccount: req.DestinationAccountReference,
		transitionsLeft: 3, terminalStatus: "SUCCEEDED",
		createdAt: time.Now().UTC(),
	}
	// Surface the in-flight payout/transfer in /v1/payments as PENDING
	// straight away so the engine's periodic FetchNextPayments picks it
	// up alongside the PollPayoutStatus workflow.
	s.upsertPaymentLocked(pollPayment(polling[id], paymentType, "PENDING"))
	return initiationResponse{Mode: "polling", PollingID: id}
}

// upsertPaymentLocked inserts or updates a payment by Reference and pushes
// it to the front of s.payments. Always bumps UpdatedAt to *now* so the
// engine's `updatedAtFrom` cursor sees it on the next poll. nil payments
// are ignored (e.g. an "intermediate" poll that returns no payment yet).
// Caller MUST hold s.mu.Lock().
func (s *store) upsertPaymentLocked(p *payment) {
	if p == nil || p.Reference == "" {
		return
	}
	now := time.Now().UTC()
	for i := range s.payments {
		if s.payments[i].Reference == p.Reference {
			s.payments[i] = *p
			s.payments[i].UpdatedAt = now
			return
		}
	}
	clone := *p
	clone.UpdatedAt = now
	s.payments = append(s.payments, clone)
}

func (s *store) pollPayout(id string) initiationResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	resp := advancePoll(s.pollingPayouts, id, "PAYOUT")
	s.upsertPaymentLocked(resp.Payment)
	return resp
}

// --- transfer mutate ---------------------------------------------------------
//
// Transfers and payouts share the same envelope but are tracked separately
// so the engine's per-primitive idempotency / polling state never leaks
// across primitives — exactly how a real PSP would behave when both
// CREATE_TRANSFER and CREATE_PAYOUT are advertised.

func (s *store) initiateTransfer(idemKey string, req initiationRequest) initiationResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.initiateTransferLocked(idemKey, req)
}

func (s *store) pollTransfer(id string) initiationResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	resp := advancePoll(s.pollingTransfers, id, "TRANSFER")
	s.upsertPaymentLocked(resp.Payment)
	return resp
}

// advancePoll is the shared poll state machine: PENDING for `transitionsLeft`
// calls, then the entry's terminal status. Lifted out of pollPayout so
// pollTransfer reuses the exact same logic.
func advancePoll(m map[string]*pollEntry, id, paymentType string) initiationResponse {
	entry, ok := m[id]
	if !ok {
		return initiationResponse{Mode: "polling", PollingID: id, Payment: nil}
	}
	if entry.transitionsLeft > 0 {
		entry.transitionsLeft--
		return initiationResponse{Mode: "polling", PollingID: id, Payment: pollPayment(entry, paymentType, "PENDING")}
	}
	return initiationResponse{Mode: "polling", PollingID: id, Payment: pollPayment(entry, paymentType, entry.terminalStatus)}
}

// addExternalAccount appends a counterparty-side account so subsequent
// /v1/external-accounts polls surface it. Used by handleCreateBankAccount
// to mimic real PSP behaviour where created bank accounts join the
// fetchable beneficiary list.
func (s *store) addExternalAccount(a account) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.externalAccounts = append(s.externalAccounts, a)
}

// resolveBankAccount returns the cached external-account reference for an
// idempotent re-POST. Returns ("", false) if this is a fresh bank-account
// creation. The caller persists via addExternalAccount + records the
// mapping under idemBankAccount.
func (s *store) resolveBankAccount(idemKey string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ref, ok := s.idemBankAccount[idemKey]
	return ref, ok
}

func (s *store) recordBankAccount(idemKey, ref string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.idemBankAccount[idemKey] = ref
}

// firstPaymentRef returns the first seeded payment's reference (used by
// the webhook trigger to surface a realistic payment.* event).
func (s *store) firstPaymentRef() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.payments) == 0 {
		return ""
	}
	return s.payments[0].Reference
}

func (s *store) firstPayment() (payment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.payments) == 0 {
		return payment{}, false
	}
	return s.payments[0], true
}

func (s *store) firstAccount() (account, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.accounts) == 0 {
		return account{}, false
	}
	return s.accounts[0], true
}

// findWebhookCallback looks up the registered callback URL for an event
// name. Returns ("", false) if no subscription is active for that event.
func (s *store) findWebhookCallback(name string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sub := range s.webhookSubs {
		if sub.Name == name {
			return sub.CallbackURL, true
		}
	}
	return "", false
}

// webhooksRegistered reports whether the engine has subscribed to any
// event topic. Drives the gating logic for poll-driven evolution: when
// webhooks are active, polls are slow heartbeats and the counterparty is
// expected to push state changes — so polls don't auto-evolve.
func (s *store) webhooksRegistered() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.webhookSubs) > 0
}

// findPayment is the read-only lookup used by the server's webhook
// auto-emission path: when EvolveSteps reports "payment X just
// transitioned", the server materialises the payload by re-fetching
// the record (post-mutation snapshot) and pushes it. There is no
// findOrder/findConversion equivalent because the universal contract
// doesn't expose order or conversion events as webhooks (the engine's
// WebhookResponse has no surface for them).
func (s *store) findPayment(ref string) (payment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.payments {
		if p.Reference == ref {
			return p, true
		}
	}
	return payment{}, false
}

func terminalPayment(r initiationRequest, paymentType string) *payment {
	now := time.Now().UTC()
	return &payment{
		Reference: r.Reference, CreatedAt: now, UpdatedAt: now,
		Type: paymentType, Status: "SUCCEEDED",
		Amount: r.Amount, Asset: r.Asset,
		SourceAccountReference: ptr(r.SourceAccountReference), DestinationAccountReference: ptr(r.DestinationAccountReference),
	}
}

func pollPayment(e *pollEntry, paymentType, status string) *payment {
	now := time.Now().UTC()
	return &payment{
		Reference: e.reference, CreatedAt: e.createdAt, UpdatedAt: now,
		Type: paymentType, Status: status,
		Amount: e.amount, Asset: e.asset,
		SourceAccountReference: ptr(e.srcAccount), DestinationAccountReference: ptr(e.dstAccount),
	}
}
