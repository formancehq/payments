package krakenpro

import (
	"encoding/json"
	"testing"
)

func TestLedgerWindowPlanFreezesEnd(t *testing.T) {
	t.Parallel()
	w := ledgerWindow{Watermark: 500}
	start, end, ofs := w.plan(1000)
	if start != 500 || end != 1000 || ofs != 0 {
		t.Fatalf("plan: start=%v end=%v ofs=%v", start, end, ofs)
	}
	if w.End != 1000 {
		t.Fatalf("End not frozen in state: %v", w.End)
	}
	// A later plan within the same drain must keep the frozen end so
	// rows arriving mid-drain can't shift ofs positions.
	_, end2, _ := w.plan(9999)
	if end2 != 1000 {
		t.Fatalf("End must stay frozen mid-drain, got %v", end2)
	}
}

func TestLedgerWindowAdvanceFullPage(t *testing.T) {
	t.Parallel()
	w := ledgerWindow{Watermark: 500, End: 1000, Offset: 0}
	if more := w.advance(50, 50); !more {
		t.Fatal("full page → hasMore=true")
	}
	if w.Offset != 50 {
		t.Fatalf("offset not advanced: %d", w.Offset)
	}
	if w.End != 1000 || w.Watermark != 500 {
		t.Fatalf("window must not promote mid-drain: %+v", w)
	}
}

func TestLedgerWindowAdvanceShortPagePromotes(t *testing.T) {
	t.Parallel()
	w := ledgerWindow{Watermark: 500, End: 1000, Offset: 100}
	if more := w.advance(10, 50); more {
		t.Fatal("short page → hasMore=false")
	}
	if w.Watermark != 1000 {
		t.Fatalf("watermark must promote to frozen end: %v", w.Watermark)
	}
	if w.End != 0 || w.Offset != 0 {
		t.Fatalf("window must reset after drain: %+v", w)
	}
}

// TestLedgerWindowFullDrainNoSkip proves the FSM drains a window larger
// than PAGE_SIZE across multiple pages, covering every position exactly
// once and terminating — the unit-level guarantee behind the
// orchestrator-level no-skip test.
func TestLedgerWindowFullDrainNoSkip(t *testing.T) {
	t.Parallel()
	const (
		total    = 127
		pageSize = 50
	)
	var w ledgerWindow
	seen := 0
	for i := 0; i < 100; i++ { // safety bound; should break well before
		_, _, ofs := w.plan(1000)
		page := total - ofs
		if page > pageSize {
			page = pageSize
		}
		if page < 0 {
			page = 0
		}
		seen += page
		if !w.advance(page, pageSize) {
			break
		}
	}
	if seen != total {
		t.Fatalf("drained %d of %d rows", seen, total)
	}
	if w.Watermark != 1000 || w.End != 0 {
		t.Fatalf("window not promoted/reset after drain: %+v", w)
	}
}

func TestPaymentsStateDecode(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"window":{"watermark":1700.5,"end":1800,"offset":50}}`)
	var s paymentsState
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if s.Window.Watermark != 1700.5 || s.Window.End != 1800 || s.Window.Offset != 50 {
		t.Fatalf("window: %+v", s.Window)
	}
}

func TestConversionsStateDecode(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"window":{"watermark":1700},"pending":{"REF-1":{"id":"L-1","refid":"REF-1","time":1700,"type":"conversion","asset":"XXBT","amount":"-0.5"}}}`)
	var s conversionsState
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("decode: %v", err)
	}
	leg, ok := s.Pending["REF-1"]
	if !ok || leg.ID != "L-1" || leg.Amount != "-0.5" {
		t.Fatalf("pending leg: %+v (ok=%v)", leg, ok)
	}
}

func TestOrdersStateDecode(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"closed":{"watermark":1700.25,"end":1800,"offset":50}}`)
	var s ordersState
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if s.Closed.Watermark != 1700.25 || s.Closed.End != 1800 || s.Closed.Offset != 50 {
		t.Fatalf("closed window: %+v", s.Closed)
	}
}
