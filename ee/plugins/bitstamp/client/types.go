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

// CryptoTransactionsResponse is the payload of POST /api/v2/crypto-transactions/.
// Main-account only; sub-account scopes hit a 4xx the orchestrator's
// try-and-skip cache handles.
type CryptoTransactionsResponse struct {
	Deposits              []CryptoDeposit        `json:"deposits"`
	Withdrawals           []CryptoWithdrawal     `json:"withdrawals"`
	RippleIOUTransactions []RippleIOUTransaction `json:"ripple_iou_transactions"`
}

// CryptoDeposit carries status ("PENDING"/"COMPLETED") + pending_reason
// (set only on PENDING). Datetime is Unix seconds, NOT the string
// format used by user_transactions.
type CryptoDeposit struct {
	ID                 int64       `json:"id"`
	Network            string      `json:"network"`
	Currency           string      `json:"currency"`
	TxID               string      `json:"txid"`
	Amount             json.Number `json:"amount"`
	Datetime           int64       `json:"datetime"`
	Status             string      `json:"status"`
	PendingReason      string      `json:"pending_reason,omitempty"`
	DestinationAddress string      `json:"destinationAddress,omitempty"`
}

// CryptoWithdrawal has no status field — the endpoint only surfaces
// processed (settled) rows.
type CryptoWithdrawal struct {
	Currency           string      `json:"currency"`
	Network            string      `json:"network"`
	DestinationAddress string      `json:"destinationAddress,omitempty"`
	TxID               string      `json:"txid"`
	Amount             json.Number `json:"amount"`
	Datetime           int64       `json:"datetime"`
}

// RippleIOUTransaction is shaped like CryptoWithdrawal.
type RippleIOUTransaction struct {
	Currency           string      `json:"currency"`
	Network            string      `json:"network"`
	DestinationAddress string      `json:"destinationAddress,omitempty"`
	TxID               string      `json:"txid"`
	Amount             json.Number `json:"amount"`
	Datetime           int64       `json:"datetime"`
}

// WithdrawalRequest is one row from POST /api/v2/withdrawal-requests/.
// type enum: 0=SEPA, 1=international wire, 2=ARDI, 3=international BIC, 4=crypto.
// status enum: 0|1=pending, 2=processed, 3=canceled, 4=failed.
type WithdrawalRequest struct {
	ID            int64  `json:"id"`
	Datetime      string `json:"datetime"`
	Type          int    `json:"type"`
	Currency      string `json:"currency"`
	Network       string `json:"network,omitempty"`
	Amount        string `json:"amount"`
	Status        int    `json:"status"`
	TxID          string `json:"txid,omitempty"`
	Address       string `json:"address,omitempty"`
	TransactionID string `json:"transaction_id,omitempty"`
}

// CryptoTransactionsOptions parameterises POST /api/v2/crypto-transactions/.
// since_timestamp / until_timestamp are bounded to 30 days; the
// orchestrator uses watermark-based pagination instead.
type CryptoTransactionsOptions struct {
	Limit          int
	Offset         int
	IncludeIOUs    bool
	SinceTimestamp int64
	UntilTimestamp int64
}

// OpenOrder is one row from POST /api/v2/open_orders/all/. Type values:
// "0" = BUY, "1" = SELL.
type OpenOrder struct {
	ID            string `json:"id"`
	ClientOrderID string `json:"client_order_id,omitempty"`
	Datetime      string `json:"datetime"`
	Type          string `json:"type"`
	Price         string `json:"price"`
	Amount        string `json:"amount"`
	CurrencyPair  string `json:"currency_pair"`
}

// OrderStatus is the response from POST /api/v2/order_status/. The
// rich shape returns market / type / subtype / datetime /
// amount_remaining live, so only the original limit Price needs
// first-sight capture from open_orders/. Derivatives-only fields
// let the spot-only mapper check HasDerivativesMarker() before mapping.
type OrderStatus struct {
	ID              json.Number        `json:"id"`
	ClientOrderID   string             `json:"client_order_id,omitempty"`
	Datetime        string             `json:"datetime,omitempty"`
	Type            string             `json:"type,omitempty"`
	Subtype         string             `json:"subtype,omitempty"`
	Status          string             `json:"status"`
	Market          string             `json:"market,omitempty"`
	AmountRemaining string             `json:"amount_remaining,omitempty"`
	Transactions    []OrderTransaction `json:"transactions"`

	MarginMode      string `json:"margin_mode,omitempty"`
	Leverage        string `json:"leverage,omitempty"`
	StopPrice       string `json:"stop_price,omitempty"`
	Trigger         string `json:"trigger,omitempty"`
	ActivationPrice string `json:"activation_price,omitempty"`
	TrailingDelta   int    `json:"trailing_delta,omitempty"`
}

func (os OrderStatus) HasDerivativesMarker() bool {
	return os.MarginMode != "" || os.Leverage != "" || os.StopPrice != "" ||
		os.Trigger != "" || os.ActivationPrice != "" || os.TrailingDelta != 0
}

// OrderTransaction is one fill on an order_status response. Carries
// the same dynamic per-currency key shape as UserTransaction.
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
		if s, ok := decodeStringDecimal(val); ok {
			ot.CurrencyAmounts[key] = s
		}
	}
	return nil
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
