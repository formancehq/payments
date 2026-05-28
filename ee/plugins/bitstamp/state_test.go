package bitstamp

import (
	"encoding/json"
	"testing"
)

// TestPaymentsStateDecode_LegacyFlatShape locks the migration path from
// the legacy single-watermark {"lastTransactionID": N} state JSON.
func TestPaymentsStateDecode_LegacyFlatShape(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		json   string
		wantID int64
	}{
		{"non-zero watermark", `{"lastTransactionID": 458254264}`, 458254264},
		{"zero watermark", `{"lastTransactionID": 0}`, 0},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var s paymentsState
			if err := json.Unmarshal([]byte(tc.json), &s); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if s.LastTransactionID != tc.wantID {
				t.Errorf("LastTransactionID = %d, want %d", s.LastTransactionID, tc.wantID)
			}
		})
	}
}

// TestPaymentsStateRoundTrip asserts the flat shape round-trips.
func TestPaymentsStateRoundTrip(t *testing.T) {
	t.Parallel()
	in := paymentsState{LastTransactionID: 999}
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

// TestPaymentsStateDecode_EmptyObject must yield the zero value (cold start).
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

// TestOrdersStateDecode_LegacyTrackedOrders: the previous ordersState had a
// "trackedOrders" map and then "lastSeenIDPerMarket". Old state must decode
// into the new shape without error; unknown fields are ignored and
// LastSeenEventIDPerMarket starts empty (cold start — correct on upgrade).
func TestOrdersStateDecode_LegacyTrackedOrders(t *testing.T) {
	t.Parallel()
	payload := `{
		"trackedOrders": {
			"100": {"lastStatus": "Open", "firstSeenAt": "2025-09-25T14:00:00Z", "limitPrice": "60000.00"}
		}
	}`
	var s ordersState
	if err := json.Unmarshal([]byte(payload), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(s.LastSeenEventIDPerMarket) != 0 {
		t.Errorf("LastSeenEventIDPerMarket must be empty on upgrade from legacy state, got %v", s.LastSeenEventIDPerMarket)
	}
}

func TestOrdersStateRoundTrip(t *testing.T) {
	t.Parallel()
	in := ordersState{
		LastSeenEventIDPerMarket: map[string]string{
			"btcusd": "a1b2c3d4-e5f6-a1b2-c3d4-e5f6a1b2c3d4",
			"ethusd": "00112233-4455-6677-8899-aabbccddeeff",
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
	if out.LastSeenEventIDPerMarket["btcusd"] != in.LastSeenEventIDPerMarket["btcusd"] ||
		out.LastSeenEventIDPerMarket["ethusd"] != in.LastSeenEventIDPerMarket["ethusd"] {
		t.Errorf("round-trip mismatch: got %+v, want %+v", out, in)
	}
}

// TestAdvanceInt64Cursor locks the canonical "never reset on empty
// response" invariant.
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

func TestIsValidMarketEventID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		id   string
		want bool
	}{
		{"000652ba-1467-f198-0000-00d800000020", true},   // valid UUID lowercase
		{"AABBCCDD-EEFF-0011-2233-445566778899", true},   // valid UUID uppercase
		{"", false},                                       // empty
		{"tooshort", false},                               // too short
		{"a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4", false},     // 32-char hex, no hyphens
		{"000652ba-1467-f198-0000-00d80000002", false},   // UUID too short by one
		{"000652ba-1467-f198-0000-00d8000000200", false}, // UUID too long by one
		{"000652ba_1467_f198_0000_00d800000020", false},  // wrong separator
		{"zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz", false}, // non-hex UUID
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.id, func(t *testing.T) {
			t.Parallel()
			if got := isValidMarketEventID(tc.id); got != tc.want {
				t.Errorf("isValidMarketEventID(%q) = %v, want %v", tc.id, got, tc.want)
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
