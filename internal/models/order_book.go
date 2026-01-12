package models

import (
	"math/big"
	"time"
)

// OrderBookEntry represents a single price level in the order book
type OrderBookEntry struct {
	Price    *big.Int `json:"price"`
	Quantity *big.Int `json:"quantity"`
}

// OrderBook represents the order book for a trading pair
type OrderBook struct {
	Pair      string           `json:"pair"`      // Trading pair (e.g., "BTC/USD")
	Bids      []OrderBookEntry `json:"bids"`      // Buy orders sorted by price descending
	Asks      []OrderBookEntry `json:"asks"`      // Sell orders sorted by price ascending
	Timestamp time.Time        `json:"timestamp"` // When the order book snapshot was taken
}

// GetOrderBookRequest contains parameters for fetching an order book
type GetOrderBookRequest struct {
	Pair  string // Trading pair (e.g., "BTC/USD")
	Depth int    // Maximum number of price levels to return (0 = connector default)
}

// GetOrderBookResponse contains the order book data
type GetOrderBookResponse struct {
	OrderBook OrderBook
}
