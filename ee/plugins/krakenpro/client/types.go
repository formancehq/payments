package client

// Envelope is the standard Kraken REST v0 wire shape:
//
//	{"error": ["EClass:Subclass", ...], "result": { ... }}
//
// A request is successful iff len(Error) == 0; even on HTTP 200 the
// error slice may carry one or more entries.
type Envelope struct {
	Error  []string `json:"error"`
	Result any      `json:"result,omitempty"`
}

// AssetInfo is one row from /0/public/Assets. The "decimals" field is
// the canonical precision used for amount conversion; "display_decimals"
// is for UI rendering and is intentionally ignored.
type AssetInfo struct {
	Altname         string `json:"altname"`
	Aclass          string `json:"aclass"`
	Decimals        int    `json:"decimals"`
	DisplayDecimals int    `json:"display_decimals"`
	Status          string `json:"status,omitempty"`
}

// AssetPair is one row from /0/public/AssetPairs.
type AssetPair struct {
	Altname      string `json:"altname"`
	Wsname       string `json:"wsname"`
	AclassBase   string `json:"aclass_base"`
	Base         string `json:"base"`
	AclassQuote  string `json:"aclass_quote"`
	Quote        string `json:"quote"`
	PairDecimals int    `json:"pair_decimals"`
	LotDecimals  int    `json:"lot_decimals"`
	CostDecimals int    `json:"cost_decimals"`
	Status       string `json:"status,omitempty"`
}

// BalanceExEntry is one row from /0/private/BalanceEx. Asset keys
// arrive as map keys at the response root. Credit / CreditUsed are
// populated on VIP/Pro accounts with a credit line; available is
// balance + credit - credit_used - hold_trade (see mappers.balance).
type BalanceExEntry struct {
	Balance    string `json:"balance"`
	HoldTrade  string `json:"hold_trade"`
	Credit     string `json:"credit"`
	CreditUsed string `json:"credit_used"`
}

// LedgerEntry is one row from /0/private/Ledgers, indexed by ledger ID.
// time is a UNIX epoch float (seconds.fractions). ID is not part of the
// wire row (it arrives as the map key); the orchestrator fills it so a
// buffered conversion leg can be persisted/replayed as a whole entry.
type LedgerEntry struct {
	ID      string  `json:"id,omitempty"`
	Refid   string  `json:"refid"`
	Time    float64 `json:"time"`
	Type    string  `json:"type"`
	Subtype string  `json:"subtype"`
	Aclass  string  `json:"aclass"`
	Asset   string  `json:"asset"`
	Amount  string  `json:"amount"`
	Fee     string  `json:"fee"`
	Balance string  `json:"balance"`
}

// LedgersResponse wraps a /0/private/Ledgers result body.
type LedgersResponse struct {
	Ledger map[string]LedgerEntry `json:"ledger"`
	Count  int                    `json:"count,omitempty"`
}

// OrderDescr is the nested `descr` block on /0/private/OpenOrders and
// ClosedOrders rows. It holds the order's defining attributes (pair,
// direction, type, prices); the cumulative execution state lives on
// the parent OrderEntry.
type OrderDescr struct {
	Pair      string `json:"pair"`             // "XBTUSD"
	Type      string `json:"type"`             // "buy" / "sell"
	Ordertype string `json:"ordertype"`        // "market" / "limit" / "stop-loss" / ...
	Price     string `json:"price"`            // limit price, or trigger price for stop-loss/take-profit orders
	Price2    string `json:"price2,omitempty"` // limit price for stop-loss-limit/take-profit-limit orders
	Leverage  string `json:"leverage,omitempty"`
	Order     string `json:"order,omitempty"` // human-readable description
}

// OrderEntry is one row from /0/private/OpenOrders or ClosedOrders. It
// carries the order's cumulative state (vol ordered, vol_exec filled,
// cost/fee), not a single fill — which is what keeps emissions aligned
// with the engine's adjustment dedup. Trades holds the per-fill txids
// when the caller passed trades:true.
type OrderEntry struct {
	Refid    string     `json:"refid"`
	ClOrdID  string     `json:"cl_ord_id,omitempty"` // client-assigned order id (when placed with one)
	Userref  any        `json:"userref,omitempty"`
	Status   string     `json:"status"` // pending / open / closed / canceled / expired
	Opentm   float64    `json:"opentm"`
	Closetm  float64    `json:"closetm,omitempty"` // closed orders only
	Starttm  float64    `json:"starttm,omitempty"`
	Expiretm float64    `json:"expiretm,omitempty"`
	Descr    OrderDescr `json:"descr"`
	Vol      string     `json:"vol"`              // ordered base quantity
	VolExec  string     `json:"vol_exec"`         // cumulative filled base quantity
	Cost     string     `json:"cost"`             // cumulative quote spend/receive
	Fee      string     `json:"fee"`              // cumulative fee (quote currency)
	Price    string     `json:"price"`            // average fill price
	Reason   string     `json:"reason,omitempty"` // cancel reason for closed/canceled
	Misc     string     `json:"misc,omitempty"`
	Oflags   string     `json:"oflags,omitempty"`
	Trades   []string   `json:"trades,omitempty"` // per-fill txids when requested
}

// ClosedOrdersResponse wraps a /0/private/ClosedOrders result body.
// `count` is the total available given the request's filter
// (omitted when `without_count: true` was set).
type ClosedOrdersResponse struct {
	Closed map[string]OrderEntry `json:"closed"`
	Count  int                   `json:"count,omitempty"`
}
