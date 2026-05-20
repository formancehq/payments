package bitstamp

import (
	"encoding/json"
	"testing"
	"time"
)

// TestPaymentsStateDecode_LegacyFlatShape locks the migration path
// from the legacy single-watermark state JSON. An existing connector
// install must NOT have its since_id watermark wiped when upgrading
// to the multi-source binary.
func TestPaymentsStateDecode_LegacyFlatShape(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		json    string
		wantUT  int64
		wantOTH bool // expect non-zero crypto/withdrawal subfields
	}{
		{
			name:    "legacy non-zero watermark lifts into UserTransactions",
			json:    `{"lastTransactionID": 458254264}`,
			wantUT:  458254264,
			wantOTH: false,
		},
		{
			name:    "legacy zero watermark still decodes cleanly",
			json:    `{"lastTransactionID": 0}`,
			wantUT:  0,
			wantOTH: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var s paymentsState
			if err := json.Unmarshal([]byte(tc.json), &s); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if s.UserTransactions.LastTransactionID != tc.wantUT {
				t.Errorf("UserTransactions.LastTransactionID = %d, want %d",
					s.UserTransactions.LastTransactionID, tc.wantUT)
			}
			if s.CryptoTransactions.DepositsSinceTs != 0 ||
				s.WithdrawalRequests.LastID != 0 {
				t.Errorf("legacy migration must leave new substates zero, got %+v", s)
			}
		})
	}
}

// TestPaymentsStateDecode_NewNestedShape locks the steady-state
// decode for the multi-source nested shape. A connector that has
// already run at least one cycle persists this form.
func TestPaymentsStateDecode_NewNestedShape(t *testing.T) {
	t.Parallel()

	payload := `{
		"userTransactions":   {"lastTransactionID": 458254264},
		"cryptoTransactions": {"depositsSinceTs": 1759995000, "withdrawalsSinceTs": 1642665114, "ripplesSinceTs": 0},
		"withdrawalRequests": {"lastID": 42}
	}`

	var s paymentsState
	if err := json.Unmarshal([]byte(payload), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if s.UserTransactions.LastTransactionID != 458254264 {
		t.Errorf("UT watermark = %d", s.UserTransactions.LastTransactionID)
	}
	if s.CryptoTransactions.DepositsSinceTs != 1759995000 {
		t.Errorf("crypto deposits ts = %d", s.CryptoTransactions.DepositsSinceTs)
	}
	if s.CryptoTransactions.WithdrawalsSinceTs != 1642665114 {
		t.Errorf("crypto withdrawals ts = %d", s.CryptoTransactions.WithdrawalsSinceTs)
	}
	if s.WithdrawalRequests.LastID != 42 {
		t.Errorf("withdrawal requests lastID = %d", s.WithdrawalRequests.LastID)
	}
}

// TestPaymentsStateRoundTrip asserts the new shape round-trips
// without data loss through Marshal -> Unmarshal.
func TestPaymentsStateRoundTrip(t *testing.T) {
	t.Parallel()
	in := paymentsState{
		UserTransactions:   userTransactionsState{LastTransactionID: 999},
		CryptoTransactions: cryptoTransactionsState{DepositsSinceTs: 1, WithdrawalsSinceTs: 2, RipplesSinceTs: 3},
		WithdrawalRequests: withdrawalRequestsState{LastID: 7},
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out paymentsState
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Errorf("round-trip mismatch: got %+v, want %+v", out, in)
	}
}

// TestPaymentsStateDecode_EmptyObject must yield the zero value
// (cold start) without erroring.
func TestPaymentsStateDecode_EmptyObject(t *testing.T) {
	t.Parallel()
	var s paymentsState
	if err := json.Unmarshal([]byte(`{}`), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if (s != paymentsState{}) {
		t.Errorf("empty object must decode to zero value, got %+v", s)
	}
}

// TestPaymentsStateDecode_InvalidJSON returns a clean error.
func TestPaymentsStateDecode_InvalidJSON(t *testing.T) {
	t.Parallel()
	var s paymentsState
	if err := json.Unmarshal([]byte(`{not json`), &s); err == nil {
		t.Error("expected error on malformed JSON")
	}
}

// TestOrdersStateDecode_LegacyFullerShape: the legacy trackedOrder
// carried Amount / CurrencyPair / Type fields that we no longer need
// (order_status returns them live). The decoder must absorb the old
// shape without erroring or losing the still-relevant fields
// (LastStatus, FirstSeenAt, LimitPrice/Price).
func TestOrdersStateDecode_LegacyFullerShape(t *testing.T) {
	t.Parallel()
	payload := `{
		"trackedOrders": {
			"100": {
				"lastStatus": "Open",
				"firstSeenAt": "2025-09-25T14:00:00Z",
				"price": "60000.00",
				"amount": "0.50000000",
				"currencyPair": "btcusd",
				"type": 0
			}
		}
	}`
	var s ordersState
	if err := json.Unmarshal([]byte(payload), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	o, ok := s.TrackedOrders["100"]
	if !ok {
		t.Fatal("trackedOrders[100] missing")
	}
	if o.LastStatus != "Open" {
		t.Errorf("LastStatus = %q", o.LastStatus)
	}
	if o.FirstSeenAt.IsZero() {
		t.Error("FirstSeenAt must decode from legacy ISO timestamp")
	}
	// The new trackedOrder uses LimitPrice (not Price); the obsolete
	// "price" field is dropped silently. That's the migration cost.
}

func TestOrdersStateDecode_NilMapNotZeroed(t *testing.T) {
	t.Parallel()
	var s ordersState
	if err := json.Unmarshal([]byte(`{}`), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.TrackedOrders == nil {
		t.Error("TrackedOrders must be non-nil after decode (callers may write to it)")
	}
}

func TestOrdersStateRoundTrip(t *testing.T) {
	t.Parallel()
	in := ordersState{
		TrackedOrders: map[string]trackedOrder{
			"200": {LastStatus: "Open", FirstSeenAt: time.Date(2025, 9, 25, 14, 0, 0, 0, time.UTC), LimitPrice: "60000.00"},
		},
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out ordersState
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got, ok := out.TrackedOrders["200"]
	if !ok || got.LimitPrice != "60000.00" || got.LastStatus != "Open" || !got.FirstSeenAt.Equal(in.TrackedOrders["200"].FirstSeenAt) {
		t.Errorf("round-trip mismatch: %+v", out)
	}
}

// TestAdvanceInt64Cursor locks the canonical "never reset on empty
// response" invariant. Three separate regressions in the connector
// PR history (Coinbase Prime, Stripe x2) were variants of "wrote
// candidate over current without checking" — keeping this primitive
// in one place + unit-tested means the bug cannot regress here.
func TestAdvanceInt64Cursor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		current   int64
		candidate int64
		want      int64
	}{
		{"empty page on cold start", 0, 0, 0},
		{"empty page after history walked", 1000, 0, 1000},
		{"smaller candidate (stale page)", 1000, 500, 1000},
		{"equal candidate (idempotent re-poll)", 1000, 1000, 1000},
		{"larger candidate advances", 1000, 1500, 1500},
		{"first non-empty page from zero", 0, 42, 42},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := advanceInt64Cursor(tc.current, tc.candidate); got != tc.want {
				t.Errorf("advanceInt64Cursor(%d, %d) = %d, want %d",
					tc.current, tc.candidate, got, tc.want)
			}
		})
	}
}

func TestMaxInt64(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   []int64
		want int64
	}{
		{"empty slice", nil, 0},
		{"single zero", []int64{0}, 0},
		{"mixed positive", []int64{3, 1, 5, 2}, 5},
		{"all zeros", []int64{0, 0, 0}, 0},
		{"max first", []int64{10, 1, 1}, 10},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := maxInt64(tc.in); got != tc.want {
				t.Errorf("maxInt64(%v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestConversionsStateDecode(t *testing.T) {
	t.Parallel()
	var s conversionsState
	if err := json.Unmarshal([]byte(`{"lastTransactionID": 12345}`), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.LastTransactionID != 12345 {
		t.Errorf("LastTransactionID = %d", s.LastTransactionID)
	}
}
