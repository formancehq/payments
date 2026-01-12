package client

import "time"

// Balance represents a Kraken balance entry
type Balance struct {
	Asset  string
	Amount string
}

// Order represents a Kraken order
type Order struct {
	RefID         string    `json:"refid"`
	UserRef       int       `json:"userref"`
	Status        string    `json:"status"`
	OpenTime      float64   `json:"opentm"`
	CloseTime     float64   `json:"closetm"`
	StartTime     float64   `json:"starttm"`
	ExpireTime    float64   `json:"expiretm"`
	Descr         OrderDesc `json:"descr"`
	Vol           string    `json:"vol"`
	VolExec       string    `json:"vol_exec"`
	Cost          string    `json:"cost"`
	Fee           string    `json:"fee"`
	Price         string    `json:"price"`
	StopPrice     string    `json:"stopprice"`
	LimitPrice    string    `json:"limitprice"`
	Trigger       string    `json:"trigger"`
	Misc          string    `json:"misc"`
	OFlags        string    `json:"oflags"`
	Trades        []string  `json:"trades"`
	ClientOrderID string    `json:"cl_ord_id,omitempty"`
}

// OrderDesc contains order description
type OrderDesc struct {
	Pair      string `json:"pair"`
	Type      string `json:"type"`      // buy or sell
	OrderType string `json:"ordertype"` // market, limit, stop-loss, etc.
	Price     string `json:"price"`     // limit price
	Price2    string `json:"price2"`    // secondary limit price
	Leverage  string `json:"leverage"`
	Order     string `json:"order"`
	Close     string `json:"close"`
}

// ListOrdersParams contains parameters for listing orders
type ListOrdersParams struct {
	Trades     bool      // Whether to include trades
	UserRef    int       // Restrict to given user reference
	Start      time.Time // Start time
	End        time.Time // End time
	Offset     int       // Result offset for pagination
	CloseTime  string    // Which time to use (open, close, both)
}

// CreateOrderRequest contains parameters for creating an order
type CreateOrderRequest struct {
	OrderType     string  `json:"ordertype"`               // market, limit, stop-loss, etc.
	Type          string  `json:"type"`                    // buy or sell
	Volume        string  `json:"volume"`                  // Order volume in base currency
	Pair          string  `json:"pair"`                    // Asset pair
	Price         string  `json:"price,omitempty"`         // Limit price for limit orders
	Price2        string  `json:"price2,omitempty"`        // Secondary price (stop-loss, take-profit)
	Trigger       string  `json:"trigger,omitempty"`       // Trigger type (last, index)
	Leverage      string  `json:"leverage,omitempty"`      // Leverage amount
	ReduceOnly    bool    `json:"reduce_only,omitempty"`   // Reduce-only flag
	StartTm       string  `json:"starttm,omitempty"`       // Scheduled start time
	ExpireTm      string  `json:"expiretm,omitempty"`      // Expiration time
	TimeInForce   string  `json:"timeinforce,omitempty"`   // GTC, IOC, GTD
	ClientOrderID string  `json:"cl_ord_id,omitempty"`     // Client order ID
	Validate      bool    `json:"validate,omitempty"`      // Validate only (don't submit)
}

// CreateOrderResponse contains the response from creating an order
type CreateOrderResponse struct {
	Description string   `json:"descr"`
	TxID        []string `json:"txid"`
}

// CancelOrderResponse contains the response from canceling an order
type CancelOrderResponse struct {
	Count   int  `json:"count"`
	Pending bool `json:"pending,omitempty"`
}

// ClosedOrdersResponse represents the response from the ClosedOrders endpoint
type ClosedOrdersResponse struct {
	Orders map[string]Order `json:"closed"`
	Count  int              `json:"count"`
}

// OpenOrdersResponse represents the response from the OpenOrders endpoint
type OpenOrdersResponse struct {
	Orders map[string]Order `json:"open"`
}

// AssetPair represents a Kraken trading pair
type AssetPair struct {
	Altname           string   `json:"altname"`
	WSName            string   `json:"wsname"`
	AClassBase        string   `json:"aclass_base"`
	Base              string   `json:"base"`
	AClassQuote       string   `json:"aclass_quote"`
	Quote             string   `json:"quote"`
	Lot               string   `json:"lot"`
	CostDecimals      int      `json:"cost_decimals"`
	PairDecimals      int      `json:"pair_decimals"`
	LotDecimals       int      `json:"lot_decimals"`
	LotMultiplier     int      `json:"lot_multiplier"`
	Fees              [][]float64 `json:"fees"`
	FeeVolumeCurrency string   `json:"fee_volume_currency"`
	MarginCall        int      `json:"margin_call"`
	MarginStop        int      `json:"margin_stop"`
	OrderMin          string   `json:"ordermin"`
	CostMin           string   `json:"costmin"`
	TickSize          string   `json:"tick_size"`
	Status            string   `json:"status"`
}

// OrderBookEntry represents a single price level in the order book
// Kraken returns [price, volume, timestamp]
type OrderBookEntry struct {
	Price     string `json:"price"`
	Volume    string `json:"volume"`
	Timestamp int64  `json:"timestamp"`
}

// OrderBookResponse represents the order book from Kraken
type OrderBookResponse struct {
	Asks []OrderBookEntry `json:"asks"`
	Bids []OrderBookEntry `json:"bids"`
}

// TickerData represents ticker data from Kraken
type TickerData struct {
	A []string `json:"a"` // ask array(<price>, <whole lot volume>, <lot volume>)
	B []string `json:"b"` // bid array(<price>, <whole lot volume>, <lot volume>)
	C []string `json:"c"` // last trade closed array(<price>, <lot volume>)
	V []string `json:"v"` // volume array(<today>, <last 24 hours>)
	P []string `json:"p"` // volume weighted average price array(<today>, <last 24 hours>)
	T []int    `json:"t"` // number of trades array(<today>, <last 24 hours>)
	L []string `json:"l"` // low array(<today>, <last 24 hours>)
	H []string `json:"h"` // high array(<today>, <last 24 hours>)
	O string   `json:"o"` // today's opening price
}

// TickerResponse represents the ticker response from Kraken
type TickerResponse struct {
	Data TickerData
}

// OHLCEntry represents a single OHLC candle from Kraken
// Kraken format: [timestamp, open, high, low, close, vwap, volume, count]
type OHLCEntryData struct {
	Timestamp int64
	Open      string
	High      string
	Low       string
	Close     string
	VWAP      string
	Volume    string
	Count     int
}

// OHLCResponse represents the OHLC response from Kraken
type OHLCResponse struct {
	Entries []OHLCEntryData
	Last    int64
}
