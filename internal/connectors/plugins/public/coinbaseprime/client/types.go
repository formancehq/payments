package client

import "time"

// ListOrdersParams contains parameters for listing orders
type ListOrdersParams struct {
	OrderStatuses []string  // Filter by order status
	ProductIDs    []string  // Filter by product ID
	OrderType     string    // Filter by order type
	OrderSide     string    // Filter by order side
	StartDate     time.Time // Start date for filtering
	EndDate       time.Time // End date for filtering
	Cursor        string    // Pagination cursor
	Limit         int       // Number of results per page
	SortDirection string    // ASC or DESC
}

// CreateOrderRequest contains parameters for creating an order
type CreateOrderRequest struct {
	ProductID     string `json:"product_id"`
	Side          string `json:"side"` // BUY or SELL
	Type          string `json:"type"` // MARKET or LIMIT
	BaseQuantity  string `json:"base_quantity,omitempty"`
	QuoteValue    string `json:"quote_value,omitempty"`
	LimitPrice    string `json:"limit_price,omitempty"`
	ClientOrderID string `json:"client_order_id,omitempty"`
	TimeInForce   string `json:"time_in_force,omitempty"`
	ExpiryTime    string `json:"expiry_time,omitempty"`
}

// Conversion represents a Coinbase Prime conversion (stablecoin exchange)
type Conversion struct {
	ID           string    `json:"id"`
	PortfolioID  string    `json:"portfolio_id"`
	WalletID     string    `json:"wallet_id"`
	SourceSymbol string    `json:"source_symbol"`
	TargetSymbol string    `json:"target_symbol"`
	SourceAmount string    `json:"source_amount"`
	TargetAmount string    `json:"target_amount"`
	Status       string    `json:"status"` // PENDING, COMPLETED, FAILED
	CreatedAt    time.Time `json:"created_at"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
}

type CreateConversionRequest struct {
	PortfolioID  string `json:"portfolio_id"`
	WalletID     string `json:"wallet_id"`
	SourceSymbol string `json:"source_symbol"`
	TargetSymbol string `json:"target_symbol"`
	Amount       string `json:"amount"`
}

type CreateConversionResponse struct {
	Conversion Conversion `json:"conversion"`
}

// OrderBookEntry represents a single price level in the order book
type OrderBookEntry struct {
	Price    string `json:"price"`
	Size     string `json:"size"`
	NumOrders int    `json:"num_orders,omitempty"`
}

// OrderBookResponse represents the order book from Coinbase
type OrderBookResponse struct {
	ProductID string           `json:"product_id"`
	Bids      []OrderBookEntry `json:"bids"`
	Asks      []OrderBookEntry `json:"asks"`
	Sequence  int64            `json:"sequence"`
	Time      time.Time        `json:"time"`
}

// Product represents a tradable product from Coinbase Exchange
type Product struct {
	ID                     string `json:"id"`
	BaseCurrency           string `json:"base_currency"`
	QuoteCurrency          string `json:"quote_currency"`
	BaseMinSize            string `json:"base_min_size"`
	BaseMaxSize            string `json:"base_max_size"`
	QuoteIncrement         string `json:"quote_increment"`
	BaseIncrement          string `json:"base_increment"`
	DisplayName            string `json:"display_name"`
	MinMarketFunds         string `json:"min_market_funds"`
	MaxMarketFunds         string `json:"max_market_funds"`
	MarginEnabled          bool   `json:"margin_enabled"`
	PostOnly               bool   `json:"post_only"`
	LimitOnly              bool   `json:"limit_only"`
	CancelOnly             bool   `json:"cancel_only"`
	Status                 string `json:"status"`
	StatusMessage          string `json:"status_message"`
	TradingDisabled        bool   `json:"trading_disabled"`
	FxStablecoin           bool   `json:"fx_stablecoin"`
	MaxSlippagePercentage  string `json:"max_slippage_percentage"`
	AuctionMode            bool   `json:"auction_mode"`
	HighBidLimitPercentage string `json:"high_bid_limit_percentage"`
}
