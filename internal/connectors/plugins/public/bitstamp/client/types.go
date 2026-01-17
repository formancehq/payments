package client

import "time"

// Balance represents a Bitstamp balance entry
type Balance struct {
	Currency         string `json:"currency"`
	Available        string `json:"available"`
	Balance          string `json:"balance"`
	Reserved         string `json:"reserved"`
	WithdrawalFee    string `json:"withdrawal_fee"`
	AvailableForSale string `json:"available_for_sale"`
}

// BalanceResponse represents the response from the balance endpoint
type BalanceResponse map[string]Balance

// AccountBalancesResponse represents the v2 account_balances response
type AccountBalancesResponse []AccountBalance

// AccountBalance represents a single account balance
type AccountBalance struct {
	Currency  string `json:"currency"`
	Available string `json:"available"`
	Balance   string `json:"total"`
	Reserved  string `json:"reserved"`
}

// Order represents a Bitstamp order
type Order struct {
	ID           string  `json:"id"`
	DateTime     string  `json:"datetime"`
	Type         string  `json:"type"` // 0 = buy, 1 = sell
	Price        string  `json:"price"`
	Amount       string  `json:"amount"`
	AmountAtCreate string `json:"amount_at_create,omitempty"`
	CurrencyPair string  `json:"currency_pair,omitempty"`
	Market       string  `json:"market,omitempty"`
	Status       string  `json:"status,omitempty"`
	ClientOrderID string `json:"client_order_id,omitempty"`
}

// OrderStatus represents order status from Bitstamp
type OrderStatus struct {
	ID              int64    `json:"id"`
	Status          string   `json:"status"` // Open, Finished, Canceled
	AmountRemaining string   `json:"amount_remaining"`
	Transactions    []Transaction `json:"transactions"`
	ClientOrderID   string   `json:"client_order_id,omitempty"`
}

// Transaction represents a trade/transaction on Bitstamp
type Transaction struct {
	TID      int64   `json:"tid"`
	Price    string  `json:"price"`
	Fee      string  `json:"fee"`
	Datetime string  `json:"datetime"`
	Type     int     `json:"type"` // 0 = deposit, 1 = withdrawal, 2 = market trade
}

// ListOrdersParams contains parameters for listing orders
type ListOrdersParams struct {
	Offset int
	Limit  int
	Since  time.Time
}

// CreateOrderRequest contains parameters for creating an order
type CreateOrderRequest struct {
	Market        string // Trading pair (e.g., "btcusd")
	Amount        string // Order amount
	Price         string // Limit price (for limit orders)
	LimitPrice    string // Limit price for stop orders
	DailyOrder    bool   // Daily order flag
	IOCOrder      bool   // Immediate or cancel
	FOKOrder      bool   // Fill or kill
	MocOrder      bool   // Market on close
	GtdOrder      bool   // Good till date
	ExpireTime    int64  // Expiration time for GTD orders
	ClientOrderID string // Client-provided order ID
}

// CreateOrderResponse contains the response from creating an order
type CreateOrderResponse struct {
	ID            string `json:"id"`
	DateTime      string `json:"datetime"`
	Type          string `json:"type"`
	Price         string `json:"price"`
	Amount        string `json:"amount"`
	ClientOrderID string `json:"client_order_id,omitempty"`
}

// CancelOrderResponse contains the response from canceling an order
type CancelOrderResponse struct {
	ID     int64  `json:"id"`
	Amount string `json:"amount"`
	Price  string `json:"price"`
	Type   int    `json:"type"`
}

// TradingPair represents a Bitstamp trading pair
type TradingPair struct {
	Name               string `json:"name"`
	URLSymbol          string `json:"url_symbol"`
	BaseDecimals       int    `json:"base_decimals"`
	CounterDecimals    int    `json:"counter_decimals"`
	InstantOrderCounter int   `json:"instant_order_counter_decimals"`
	MinimumOrder       string `json:"minimum_order"`
	Trading            string `json:"trading"` // "Enabled" or "Disabled"
	Description        string `json:"description"`
}

// OrderBookEntry represents a single price level in the order book
type OrderBookEntry struct {
	Price  string
	Amount string
}

// OrderBookResponse represents the order book from Bitstamp
type OrderBookResponse struct {
	Timestamp string     `json:"timestamp"`
	Bids      [][]string `json:"bids"` // [[price, amount], ...]
	Asks      [][]string `json:"asks"` // [[price, amount], ...]
}

// TickerResponse represents ticker data from Bitstamp
type TickerResponse struct {
	Last      string `json:"last"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Vwap      string `json:"vwap"`
	Volume    string `json:"volume"`
	Bid       string `json:"bid"`
	Ask       string `json:"ask"`
	Timestamp string `json:"timestamp"`
	Open      string `json:"open"`
	Open24    string `json:"open_24"`
	PercentChange24 string `json:"percent_change_24"`
}

// OHLCEntry represents a single OHLC candle from Bitstamp
type OHLCEntry struct {
	Timestamp int64  `json:"timestamp"`
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
	Volume    string `json:"volume"`
}

// OHLCResponse represents the OHLC response from Bitstamp
type OHLCResponse struct {
	Data struct {
		OHLC []OHLCEntry `json:"ohlc"`
		Pair string      `json:"pair"`
	} `json:"data"`
}

// UserTransaction represents a user transaction from Bitstamp
type UserTransaction struct {
	ID       int64  `json:"id"`
	DateTime string `json:"datetime"`
	Type     string `json:"type"` // 0 = deposit, 1 = withdrawal, 2 = market trade, 14 = sub account transfer
	Fee      string `json:"fee"`
	OrderID  int64  `json:"order_id"`
}

// InstantOrderRequest contains parameters for creating an instant (market) order
// Used for conversions - executing immediately at current market price
type InstantOrderRequest struct {
	Market        string // Trading pair (e.g., "btcusd")
	Amount        string // Amount to buy/sell
	ClientOrderID string // Optional client-provided order ID
}

// InstantOrderResponse contains the response from an instant order
type InstantOrderResponse struct {
	ID            string `json:"id"`
	DateTime      string `json:"datetime"`
	Type          string `json:"type"` // 0 = buy, 1 = sell
	Price         string `json:"price"`
	Amount        string `json:"amount"`
	ClientOrderID string `json:"client_order_id,omitempty"`
}
