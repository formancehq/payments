package client

import (
	"encoding/json"
	"math/big"
	"strings"
)

// AccountBalance is one row from POST /api/v2/account_balances/.
// One row per supported currency regardless of holdings; zero-balance
// rows are filtered at the accounts mapper.
type AccountBalance struct {
	Currency  string `json:"currency"`
	Total     string `json:"total"`
	Available string `json:"available"`
	Reserved  string `json:"reserved"`
}

// UserTransaction is one row from POST /api/v2/user_transactions/.
// Per-currency amounts arrive as dynamic top-level string keys
// (e.g. "btc", "eur") and pair-rate keys like "usdc_eur" arrive
// alongside; both are extracted via UnmarshalJSON into CurrencyAmounts
// and PairRates respectively.
type UserTransaction struct {
	ID             int64       `json:"id"`
	Datetime       string      `json:"datetime"`
	Type           string      `json:"type"`
	Fee            string      `json:"fee"`
	OrderID        json.Number `json:"order_id,omitempty"`
	SelfTrade      bool        `json:"self_trade,omitempty"`
	SelfTradeOrder json.Number `json:"self_trade_order_id,omitempty"`

	// Derivatives markers — spot-only mapper skips rows that carry these.
	MarginMode   string `json:"margin_mode,omitempty"`
	LeverageRate string `json:"leverage_rate,omitempty"`

	CurrencyAmounts map[string]string
	PairRates       map[string]string
}

func (ut UserTransaction) HasDerivativesMarker() bool {
	return ut.MarginMode != "" || ut.LeverageRate != ""
}

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
		// Pair rate keys carry an underscore (<src>_<dst>) and may
		// arrive as a string or a number depending on firmware.
		if strings.IndexByte(key, '_') > 0 {
			if s, ok := decodeDecimal(val); ok {
				ut.PairRates[key] = s
			}
			continue
		}
		// Currency amounts must be string decimals — numeric extras
		// (e.g. a future numeric field) must not be mistaken for a
		// phantom currency.
		if s, ok := decodeStringDecimal(val); ok {
			ut.CurrencyAmounts[key] = s
		}
	}
	return nil
}

// Currency describes a Bitstamp-listed asset with its decimal
// precision + per-blockchain network rollup.
type Currency struct {
	Name            string            `json:"name"`
	Currency        string            `json:"currency"`
	Decimals        int               `json:"decimals"`
	Type            string            `json:"type"`
	Symbol          string            `json:"symbol,omitempty"`
	Logo            string            `json:"logo,omitempty"`
	AvailableSupply string            `json:"available_supply,omitempty"`
	Deposit         string            `json:"deposit,omitempty"`
	Withdrawal      string            `json:"withdrawal,omitempty"`
	Networks        []CurrencyNetwork `json:"networks,omitempty"`
}

type CurrencyNetwork struct {
	Network                 string `json:"network"`
	Deposit                 string `json:"deposit,omitempty"`
	Withdrawal              string `json:"withdrawal,omitempty"`
	WithdrawalDecimals      int    `json:"withdrawal_decimals,omitempty"`
	WithdrawalMinimumAmount string `json:"withdrawal_minimum_amount,omitempty"`
}

// Market is one row from GET /api/v2/markets/.
type Market struct {
	BaseCurrency                string `json:"base_currency"`
	BaseDecimals                int    `json:"base_decimals"`
	CounterCurrency             string `json:"counter_currency"`
	CounterDecimals             int    `json:"counter_decimals"`
	Description                 string `json:"description,omitempty"`
	InstantAndMarketOrders      string `json:"instant_and_market_orders,omitempty"`
	InstantOrderCounterDecimals int    `json:"instant_order_counter_decimals,omitempty"`
	MarketSymbol                string `json:"market_symbol"`
	MarketType                  string `json:"market_type,omitempty"`
	MinimumOrderValue           string `json:"minimum_order_value,omitempty"`
	Name                        string `json:"name,omitempty"`
	Trading                     string `json:"trading,omitempty"`
}

// MyMarket is one row from GET /api/v2/my_markets/ (signed).
type MyMarket struct {
	Name      string `json:"name"`
	URLSymbol string `json:"url_symbol"`
}

// TradingFee is one row from POST /api/v2/fees/trading/.
type TradingFee struct {
	CurrencyPair string         `json:"currency_pair"`
	Market       string         `json:"market"`
	Fees         TradingFeeRate `json:"fees"`
}

// TradingFeeRate is maker/taker in string-decimal percent (e.g. "0.300").
type TradingFeeRate struct {
	Maker string `json:"maker"`
	Taker string `json:"taker"`
}

// WithdrawalFee is one row per (currency, network) — multi-chain
// assets have one row per supported network.
type WithdrawalFee struct {
	Currency string `json:"currency"`
	Fee      string `json:"fee"`
	Network  string `json:"network,omitempty"`
}

// AccountOrderDataEvent is one item from GET /api/v2/account_order_data/.
// event is "order_created" or "order_deleted"; further lifecycle events
// follow the same shape and are handled generically.
type AccountOrderDataEvent struct {
	Event          string               `json:"event"`
	EventID        string               `json:"event_id"`
	OrderSource    string               `json:"order_source"`
	TradeAccountID json.Number          `json:"trade_account_id"`
	Data           AccountOrderDataItem `json:"data"`
}

// AccountOrderDataItem is the order snapshot within an AccountOrderDataEvent.
// PriceStr may arrive in scientific notation (e.g. "7.74E+4"); callers must
// normalise it before decimal parsing.
type AccountOrderDataItem struct {
	ID             json.Number `json:"id"`
	IDStr          string      `json:"id_str"`
	OrderType      int         `json:"order_type"` // 0=BUY, 1=SELL
	OrderSubtype   int         `json:"order_subtype"`
	Datetime       string      `json:"datetime"`       // Unix seconds as string
	Microtimestamp string      `json:"microtimestamp"` // Unix microseconds as string
	Amount         json.Number `json:"amount"`
	AmountStr      string      `json:"amount_str"` // remaining amount
	AmountTraded   string      `json:"amount_traded"`
	AmountAtCreate string      `json:"amount_at_create"`
	Price          json.Number `json:"price"`
	PriceStr       string      `json:"price_str"` // may be scientific notation
	IsLiquidation  bool        `json:"is_liquidation"`
	TrailingDelta  int         `json:"trailing_delta"`
}

// decodeStringDecimal accepts only string decimal values. Numeric
// JSON values are rejected to prevent a future numeric field from
// being mistaken for a per-currency amount.
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
// number — reserved for fields where both forms are documented
// (pair rate keys today).
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
