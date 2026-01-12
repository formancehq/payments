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
