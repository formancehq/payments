package client

import (
	"encoding/json"
	"math/big"
	"strings"
)

// AccountBalance is one row from POST /api/v2/account_balances/.
// Bitstamp returns one row per supported currency, whether or not the
// user has ever held a position in it; the connector filters all-zero
// rows at the accounts mapper.
type AccountBalance struct {
	Currency  string `json:"currency"`
	Total     string `json:"total"`
	Available string `json:"available"`
	Reserved  string `json:"reserved"`
}

// UserTransaction is one row from POST /api/v2/user_transactions/.
// Per-currency amounts arrive as dynamic top-level string keys
// (e.g. "btc", "eur") — they are extracted into CurrencyAmounts via
// UnmarshalJSON. Pair rate keys like "usdc_eur" carry the rate of an
// instant buy/sell and are surfaced in PairRates so the conversion
// mapper has the rate without re-parsing Raw.
type UserTransaction struct {
	ID             int64       `json:"id"`
	Datetime       string      `json:"datetime"`
	Type           string      `json:"type"`
	Fee            string      `json:"fee"`
	OrderID        json.Number `json:"order_id,omitempty"`
	SelfTrade      bool        `json:"self_trade,omitempty"`
	SelfTradeOrder json.Number `json:"self_trade_order_id,omitempty"`
	// Derivatives markers — present only on derivatives-account rows;
	// the spot-only mapper inspects these to skip + Warn rather than
	// silently mis-classifying. See MAPPINGS.md §8.
	MarginMode    string `json:"margin_mode,omitempty"`
	LeverageRate  string `json:"leverage_rate,omitempty"`

	CurrencyAmounts map[string]string
	PairRates       map[string]string
}

// HasDerivativesMarker reports whether the row carries any of the
// known derivatives-specific fields. Used by mappers to enforce the
// spot-only stance.
func (ut UserTransaction) HasDerivativesMarker() bool {
	return ut.MarginMode != "" || ut.LeverageRate != ""
}

// userTxKnownKeys are the documented top-level keys on user_transactions
// rows; anything else is either a per-currency amount or a pair-rate
// extra. The derivatives markers (`margin_mode`, `leverage_rate`) are
// listed so they end up in neither map — the mapper inspects them via
// Raw and skips the row at Warn (spot-only stance).
var userTxKnownKeys = map[string]struct{}{
	"id":                  {},
	"datetime":            {},
	"type":                {},
	"fee":                 {},
	"order_id":            {},
	"self_trade":          {},
	"self_trade_order_id": {},
	"margin_mode":         {},
	"leverage_rate":       {},
}

func (ut *UserTransaction) UnmarshalJSON(data []byte) error {
	type alias UserTransaction
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*ut = UserTransaction(a)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	ut.CurrencyAmounts = make(map[string]string)
	ut.PairRates = make(map[string]string)
	for key, val := range raw {
		if _, known := userTxKnownKeys[key]; known {
			continue
		}
		// Pair rate keys are <src>_<dst> (lowercase). They may arrive
		// as string or number depending on Bitstamp firmware; both are
		// canonicalised to a decimal string.
		if strings.IndexByte(key, '_') > 0 {
			if s, ok := decodeDecimal(val); ok {
				ut.PairRates[key] = s
			}
			continue
		}
		// Currency amounts MUST be string decimals. Numeric values
		// (e.g. a future "created_at" timestamp) are silently dropped
		// rather than mis-parsed as a phantom currency — see PR #679
		// review item 11. Pair rates above are the documented exception.
		if s, ok := decodeStringDecimal(val); ok {
			ut.CurrencyAmounts[key] = s
		}
	}
	return nil
}

// Currency describes a Bitstamp-listed currency with its decimal
// precision. Loaded once at install and refreshed every 24h.
type Currency struct {
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Decimals int    `json:"decimals"`
	Type     string `json:"type"`
}

// OpenOrder is one row from POST /api/v2/open_orders/all/. The /all/
// variant returns currency_pair on each row; per-pair variants
// (POST /api/v2/open_orders/{pair}/) omit it because the pair is in
// the URL.
type OpenOrder struct {
	ID            string `json:"id"`
	ClientOrderID string `json:"client_order_id,omitempty"`
	Datetime      string `json:"datetime"`
	Type          string `json:"type"` // "0" = BUY, "1" = SELL
	Price         string `json:"price"`
	Amount        string `json:"amount"`
	CurrencyPair  string `json:"currency_pair"`
}

// OrderStatus is the response from POST /api/v2/order_status/. The
// payload does NOT carry the original price / amount / type /
// currency_pair — only status and the list of fills. Original
// parameters MUST be captured from open_orders/ on first sight and
// persisted on ordersState.TrackedOrders[id].
type OrderStatus struct {
	ID            json.Number        `json:"id"`
	ClientOrderID string             `json:"client_order_id,omitempty"`
	Status        string             `json:"status"`
	Transactions  []OrderTransaction `json:"transactions"`
}

// OrderTransaction is one fill on an order_status response. Same
// dynamic per-currency key shape as UserTransaction, but a narrower
// known-key set.
type OrderTransaction struct {
	TID             int64  `json:"tid"`
	Type            int    `json:"type"`
	Datetime        string `json:"datetime"`
	Price           string `json:"price"`
	Fee             string `json:"fee"`
	CurrencyAmounts map[string]string
}

var orderTxKnownKeys = map[string]struct{}{
	"tid":      {},
	"type":     {},
	"datetime": {},
	"price":    {},
	"fee":      {},
}

func (ot *OrderTransaction) UnmarshalJSON(data []byte) error {
	type alias OrderTransaction
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*ot = OrderTransaction(a)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	ot.CurrencyAmounts = make(map[string]string)
	for key, val := range raw {
		if _, known := orderTxKnownKeys[key]; known {
			continue
		}
		// Same string-only rule as UserTransaction (see PR #679 review
		// item 11): numeric extras are not assumed to be currencies.
		if s, ok := decodeStringDecimal(val); ok {
			ot.CurrencyAmounts[key] = s
		}
	}
	return nil
}

// decodeStringDecimal accepts only string decimal values. Numeric JSON
// values are rejected so a future numeric field cannot be mistaken for
// a per-currency amount.
func decodeStringDecimal(val json.RawMessage) (string, bool) {
	var s string
	if err := json.Unmarshal(val, &s); err != nil || s == "" {
		return "", false
	}
	if _, ok := new(big.Float).SetString(s); !ok {
		return "", false
	}
	return s, true
}

// decodeDecimal accepts a JSON value that is either a string or a
// number and returns its canonical decimal string. Reserved for fields
// where both forms are documented (pair rate keys today).
func decodeDecimal(val json.RawMessage) (string, bool) {
	if s, ok := decodeStringDecimal(val); ok {
		return s, true
	}
	var n json.Number
	if err := json.Unmarshal(val, &n); err == nil {
		if _, ok := new(big.Float).SetString(n.String()); ok {
			return n.String(), true
		}
	}
	return "", false
}
