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

func nowEpoch() float64 { return float64(time.Now().UnixNano()) / 1e9 }

// paymentsState carries the pagination window.
type paymentsState struct {
	Window ledgerWindow `json:"window"`
}

// conversionsState carries the pagination window + the half-paired
// buffer. Pending stores half-paired conversion rows by refid (the whole
// client.LedgerEntry, ID included) so a future page/cycle can complete
// them when the second leg arrives. Only known-asset legs are buffered,
// and entries are pruned once the watermark passes their time, so the
// map can't grow unbounded.
type conversionsState struct {
	Window  ledgerWindow                  `json:"window"`
	Pending map[string]client.LedgerEntry `json:"pending,omitempty"`
}

// ordersState is the pagination state for FetchNextOrders: ClosedOrders
// pages through the shared frozen-end + ofs window on close time.
type ordersState struct {
	Closed ledgerWindow `json:"closed"`
}
