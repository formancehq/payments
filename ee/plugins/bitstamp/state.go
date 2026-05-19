package bitstamp

import "time"

// orderRetentionMax bounds how long a tracked order may stay in
// ordersState.TrackedOrders without being terminal. Bitstamp's
// order_status endpoint retains data for ~30 days; the connector
// keeps a 5-day safety margin so an entry approaching the limit
// emits a final PSPOrder (with com.bitstamp.spec/retention_expired)
// before the data becomes unfetchable. See MAPPINGS.md §3.4.4.
const orderRetentionMax = 25 * 24 * time.Hour

// paymentsState carries the since_id watermark on the
// /api/v2/user_transactions/ stream for the payments task. Bitstamp's
// since_id is inclusive; the watermark is the highest tx.ID seen and
// is never reset at end-of-pagination (PR #707).
type paymentsState struct {
	LastTransactionID int64 `json:"lastTransactionID"`
}

// trackedOrder captures the order parameters at first sight from
// /api/v2/open_orders/all/. order_status/ does NOT return the
// original price / amount / type / currency_pair — without first-sight
// capture the connector cannot reconstruct the order intent once it
// becomes terminal. See MAPPINGS.md §3.4 + §6.2.
type trackedOrder struct {
	LastStatus   string    `json:"lastStatus"`
	FirstSeenAt  time.Time `json:"firstSeenAt"`
	Price        string    `json:"price"`
	Amount       string    `json:"amount"`
	CurrencyPair string    `json:"currencyPair"`
	Type         int       `json:"type"` // 0 = BUY, 1 = SELL
}

// ordersState carries:
//   - LastTransactionID — reserved for future user_transactions-driven
//     fill aggregation; current implementation derives fills from
//     order_status.transactions[], but the field is present so a
//     follow-up extension is non-breaking on the state JSON shape.
//   - TrackedOrders — the open-order ID → first-sight params map; the
//     orders task uses it to reconcile previously-seen orders with
//     order_status/ on every cycle and to evict at orderRetentionMax.
type ordersState struct {
	LastTransactionID int64                   `json:"lastTransactionID"`
	TrackedOrders     map[string]trackedOrder `json:"trackedOrders"`
}

// conversionsState mirrors paymentsState — same user_transactions
// stream, independent watermark so the payments and conversions
// tasks scan at potentially different cadences without contention.
type conversionsState struct {
	LastTransactionID int64 `json:"lastTransactionID"`
}
