package krakenpro

import (
	"time"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
)

// accountsState tracks the set of asset symbols already emitted as
// PSPAccounts so subsequent FetchNextAccounts cycles don't re-emit
// them. The map value stores an RFC3339 timestamp purely for
// debuggability.
type accountsState struct {
	AccountAssetsImportedAt map[string]string `json:"accountAssetsImportedAt,omitempty"`
}

// ledgerWindow paginates a newest-first Kraken stream (Ledgers,
// ClosedOrders). It freezes End at drain start, then walks ofs within
// (Watermark, End] until a short page and promotes Watermark=End. The
// frozen End fences out rows arriving mid-drain, so ofs positions stay
// stable; correctness is positional, which makes it terminate, never
// skip, and stay immune to equal-timestamp pages. This is why we use
// ofs (deprecated but positional) over the spec's ID cursor: Kraken's
// responses are unordered maps, so a page's boundary ID is unknowable.
// See MAPPINGS §3.
type ledgerWindow struct {
	Watermark float64 `json:"watermark,omitempty"` // committed exclusive lower-bound timestamp
	End       float64 `json:"end,omitempty"`       // inclusive upper bound, frozen at drain start; 0 = idle
	Offset    int     `json:"offset,omitempty"`    // next ofs within the frozen window
}

// plan freezes a new window end when idle and returns the request
// bounds (start exclusive, end inclusive, ofs page position).
func (w *ledgerWindow) plan(now float64) (start, end float64, ofs int) {
	if w.End == 0 {
		w.End = now
		w.Offset = 0
	}
	return w.Watermark, w.End, w.Offset
}

// advance records a drained page and returns hasMore. A short page
// means the frozen window is drained: promote the watermark to the
// frozen end and reset for the next cycle.
func (w *ledgerWindow) advance(pageLen, pageSize int) (hasMore bool) {
	if pageLen >= pageSize {
		w.Offset += pageLen
		return true
	}
	w.Watermark = w.End
	w.End = 0
	w.Offset = 0
	return false
}

// draining reports whether a frozen window is mid-drain (log field).
func (w ledgerWindow) draining() bool { return w.End != 0 }

// nowEpoch is the window freeze instant. Whole seconds suffice: a row
// in the freeze second with a fractional time > the floor is caught by
// the next window (start is exclusive on the same value).
func nowEpoch() float64 { return float64(time.Now().Unix()) }

// paymentsState carries the pagination window.
type paymentsState struct {
	Window ledgerWindow `json:"window"`
}

// conversionsState carries the pagination window + the half-paired
// buffer. Pending stores half-paired conversion rows by refid so a
// future page/cycle can complete them when the second leg arrives.
type conversionsState struct {
	Window  ledgerWindow          `json:"window"`
	Pending map[string]pendingLeg `json:"pending,omitempty"`
}

// pendingLeg holds the data needed to materialise one side of a
// conversion when the other side arrives in a future page/cycle.
type pendingLeg struct {
	LedgerID string  `json:"ledgerID"`
	Time     float64 `json:"time"`
	Type     string  `json:"type"`
	Subtype  string  `json:"subtype"`
	Aclass   string  `json:"aclass"`
	Asset    string  `json:"asset"`
	Amount   string  `json:"amount"` // raw signed decimal as returned by Kraken
	Fee      string  `json:"fee"`
	Balance  string  `json:"balance"`
}

// toLedgerEntry rehydrates the original client.LedgerEntry shape so
// the conversion mapper can treat carry-over legs uniformly with
// fresh ones — keeps client wire-field knowledge out of the
// orchestrator.
func (p pendingLeg) toLedgerEntry(refid string) client.LedgerEntry {
	return client.LedgerEntry{
		Refid:   refid,
		Time:    p.Time,
		Type:    p.Type,
		Subtype: p.Subtype,
		Aclass:  p.Aclass,
		Asset:   p.Asset,
		Amount:  p.Amount,
		Fee:     p.Fee,
		Balance: p.Balance,
	}
}

// ordersState is the resumable cursor for FetchNextOrders. ClosedOrders
// pages through the shared frozen-end + ofs window on close time.
//
// OpenOrders is drained in-process via Kraken's `with_cursor` paging.
// OpenCursor is normally empty (each cycle re-drains the snapshot from
// the start, which is idempotent — the engine dedupes by reference +
// status + baseFilled + fee). It is only set when a drain hits the
// in-process safety cap, so the next cycle resumes from where it stopped
// instead of restarting at page 1 and starving the tail.
type ordersState struct {
	Closed     ledgerWindow `json:"closed"`
	OpenCursor string       `json:"openCursor,omitempty"`
}
