package main

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

// store is the single-tenant, in-memory data layer. One sync.RWMutex
// covers everything (fine for the fixture; a per-tenant struct will
// replace this when multi-tenancy lands).
//
// Adjustment evolution is driven by per-record "lanes" — pre-planned
// status sequences. EvolveSteps pops the next entry off each lane,
// bumps updatedAt, and the engine sees a fresh adjustment on the next
// poll.
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

	// Per-record evolution queues. Round-robin cursors below spread
	// progress evenly across the dataset.
	paymentLanes    map[string][]string
	orderLanes      map[string][]orderStep
	conversionLanes map[string][]string

	evolvePaymentCursor    int
	evolveOrderCursor      int
	evolveConversionCursor int

	// Idempotency dedup by Idempotency-Key — one map per resource
	// family keeps the store readable.
	idemPayout       map[string]string
	idemTransfer     map[string]string
	idemBankAccount  map[string]string
	idemWebhookSubID map[string]string

	pollingPayouts   map[string]*pollEntry // pollingID -> entry
	pollingTransfers map[string]*pollEntry
	webhookSubs      map[string]webhookSub // sub.ID -> sub
	// webhookSubSeq is a monotonic counter so two installs subscribing
	// to the same event name get distinct sub_<name>_<n> ids and don't
	// overwrite each other in webhookSubs.
	webhookSubSeq int
}

// orderStep is one transition in an order's lane: target status + fill
// expressed as percent-of-ordered (0..100). The percentage keeps the
// lane definition human-readable; runtime computes minor-unit fill.
type orderStep struct {
	status  string
	fillPct int
}

// Seed sizes tuned so the engine's PAGE_SIZE (100) needs at least two
// pages on the largest endpoint — exercises cursor pagination end-to-end.
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

// paymentLaneTemplates: every trajectory the engine's adjustment table
// can produce, rotated across the seed so each lane runs somewhere.
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

// orderLaneTemplates: every OrderStatus terminal + repeated
// PARTIALLY_FILLED with growing fills (exercises the engine's per-fill
// adjustment dedup which keys on BaseQuantityFilled).
var orderLaneTemplates = [][]orderStep{
	{{"OPEN", 0}, {"PARTIALLY_FILLED", 25}, {"PARTIALLY_FILLED", 50}, {"PARTIALLY_FILLED", 75}, {"FILLED", 100}},
	{{"OPEN", 0}, {"FILLED", 100}},
	{{"OPEN", 0}, {"PARTIALLY_FILLED", 33}, {"CANCELLED", 33}},
	{{"OPEN", 0}, {"EXPIRED", 0}},
	{{"FAILED", 0}},
}

// conversionLaneTemplates: single-step terminals (conversions are
// latest-wins, no adjustment array).
var conversionLaneTemplates = [][]string{
	{"COMPLETED"},
	{"FAILED"},
}

// seed populates deterministic fixtures with strictly-increasing
// CreatedAt/UpdatedAt so the engine's `updatedAtFrom` cursor behaves
// like it would against a real PSP.
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

	// Every payment / order / conversion starts PENDING and gets a
	// lane round-robin from the catalogue above.
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
			Reference: oref,
			CreatedAt: t,
			UpdatedAt: t,
			Direction: directions[i%len(directions)],
			Type:      "MARKET",
			Status:    "PENDING",
			// MARKET is canonically IOC. Empty maps to
			// TIME_IN_FORCE_UNKNOWN whose Value() returns (nil, err) —
			// the engine's `time_in_force NOT NULL` column then fails
			// the INSERT.
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

// EvolveResult identifies one advanced record so the caller can fan a
// matching webhook event out when a subscription exists.
type EvolveResult struct {
	Kind      string // "payment" | "order" | "conversion"
	Reference string
}

// EvolveSteps advances up to `n` records, rotating across primitive
// kinds so a single call exercises every state machine in parallel.
// Each record gets a strictly-monotonic UpdatedAt (microsecond cursor)
// so the engine's `updatedAtFrom` never skips an adjustment.
//
// Driven by: the auto-evolve ticker, POST /_admin/evolve, or each
// paginated GET when evolveOnPoll is on AND no webhooks are registered.
// Safe to call concurrently.
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

// timeCursor emits microsecond-spaced monotonic timestamps so one
// EvolveSteps batch never collides on UpdatedAt.
type timeCursor struct{ t time.Time }

func newTimeCursor(t time.Time) *timeCursor { return &timeCursor{t: t} }

func (c *timeCursor) next() time.Time {
	c.t = c.t.Add(time.Microsecond)
	return c.t
}

// evolveOnePaymentLocked pops one status off the next non-empty lane
// (cursor-driven round-robin). Returns ("", false) when every lane is
// drained.
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

// evolveOneOrderLocked pops one orderStep and computes the resulting
// BaseQuantityFilled in minor units (percent-of-ordered).
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
// conversion lane.
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

// pollEntry is the per-initiation state machine: PENDING for
// `transitionsLeft` polls, then `terminalStatus`. Exercises Temporal's
// PollPayoutStatus / PollTransferStatus workflows.
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

// paginate slices `items` into one page using either the base64-encoded
// integer cursor or a 1-based page number. pageSize 0 falls back to 100
// (engine default PAGE_SIZE).
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
	filtered := s.conversions
	if !opts.updatedAtFrom.IsZero() {
		filtered = filtered[:0:0]
		for _, c := range s.conversions {
			// Conversions only carry CreatedAt on the wire — see
			// contract/data-model.md.
			if c.CreatedAt.After(opts.updatedAtFrom) {
				filtered = append(filtered, c)
			}
		}
	}
	return paginate(filtered, opts.cursor, opts.page, opts.pageSize)
}

func (s *store) listOthers(name string, opts listOpts) ([]other, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return paginate(s.others[name], opts.cursor, opts.page, opts.pageSize)
}

// filterAccountsByUpdatedAt — accounts only carry CreatedAt on the
// wire; contract uses CreatedAt as freshness anchor (append-only).
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

// initiatePayout routes ≤ €100 → terminal, > €100 → polling. The
// synthesized Payment is upserted into s.payments so the next
// /v1/payments poll surfaces it — mirrors how a real PSP exposes
// payouts as transactions. Reference matches the initiation reference
// so engine adjustments correlate.
func (s *store) initiatePayout(idemKey string, req initiationRequest) initiationResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.initiateLocked(idemKey, req, "PAYOUT", s.idemPayout, s.pollingPayouts, "ppayout_")
}

func (s *store) initiateTransferLocked(idemKey string, req initiationRequest) initiationResponse {
	return s.initiateLocked(idemKey, req, "TRANSFER", s.idemTransfer, s.pollingTransfers, "ptransfer_")
}

// initiateLocked — shared payout/transfer impl. Caller MUST hold s.mu.
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

	if mustParseInt(req.Amount) <= 10000 {
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
	// Surface the in-flight payout in /v1/payments as PENDING so the
	// next FetchNextPayments picks it up alongside Poll*.
	s.upsertPaymentLocked(pollPayment(polling[id], paymentType, "PENDING"))
	return initiationResponse{Mode: "polling", PollingID: id}
}

// upsertPaymentLocked inserts or replaces by Reference, bumps UpdatedAt
// to now so `updatedAtFrom` sees it next poll. Caller MUST hold s.mu.
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

// Transfers share the envelope but track idempotency / polling in
// their own maps so per-primitive state never leaks across primitives.
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

// advancePoll: returns Payment=nil for `transitionsLeft` calls
// ("still polling"), then a terminal Payment with terminalStatus.
//
// Contract: the engine's poll-payout workflow treats `Payment != nil`
// as a terminal result (see internal/connectors/engine/workflow/poll_payout.go).
// Returning a PENDING payment during the transition window would
// trick the engine into committing the polling workflow on the very
// first poll with a payment still in PENDING — which is exactly the
// race we want to test the engine against.
func advancePoll(m map[string]*pollEntry, id, paymentType string) initiationResponse {
	entry, ok := m[id]
	if !ok {
		return initiationResponse{Mode: "polling", PollingID: id, Payment: nil}
	}
	if entry.transitionsLeft > 0 {
		entry.transitionsLeft--
		return initiationResponse{Mode: "polling", PollingID: id, Payment: nil}
	}
	return initiationResponse{Mode: "polling", PollingID: id, Payment: pollPayment(entry, paymentType, entry.terminalStatus)}
}

// createBankAccountLocked is the atomic dedup-or-create primitive used
// by handleCreateBankAccount. Returns the resulting external account
// and whether the call deduplicated against a prior Idempotency-Key.
// Holding s.mu across the check + append eliminates the race where
// two concurrent POSTs with the same key both inserted.
func (s *store) createBankAccountLocked(idemKey, id, name, asset string) (account, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.idemBankAccount[idemKey]; ok {
		return account{
			Reference:    existing,
			CreatedAt:    time.Now().UTC(),
			Name:         &name,
			DefaultAsset: &asset,
		}, true
	}
	ref := "acct_ext_ba_" + id
	created := account{
		Reference:    ref,
		CreatedAt:    time.Now().UTC(),
		Name:         &name,
		DefaultAsset: &asset,
	}
	s.externalAccounts = append(s.externalAccounts, created)
	s.idemBankAccount[idemKey] = ref
	return created, false
}

// firstPaymentRef — first seeded payment reference (admin trigger).
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

// findWebhookCallback — registered callback URL for an event, or "" / false.
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

// webhooksRegistered gates poll-driven evolution: with webhooks on,
// polls are heartbeats and the counterparty pushes state changes.
func (s *store) webhooksRegistered() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.webhookSubs) > 0
}

// findPayment — post-mutation snapshot for webhook auto-emission. No
// order/conversion equivalent: contract has no webhook for them.
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
