package models

import (
	"math/big"
	"time"
)

// Ticker represents current market ticker information for a trading pair
type Ticker struct {
	Pair        string    `json:"pair"`        // Trading pair (e.g., "BTC/USD")
	LastPrice   *big.Int  `json:"lastPrice"`   // Last traded price
	BidPrice    *big.Int  `json:"bidPrice"`    // Best bid price
	AskPrice    *big.Int  `json:"askPrice"`    // Best ask price
	Volume24h   *big.Int  `json:"volume24h"`   // 24-hour trading volume
	High24h     *big.Int  `json:"high24h"`     // 24-hour high price
	Low24h      *big.Int  `json:"low24h"`      // 24-hour low price
	PriceChange *big.Int  `json:"priceChange"` // Price change (absolute)
	OpenPrice   *big.Int  `json:"openPrice"`   // Opening price (24h)
	Timestamp   time.Time `json:"timestamp"`   // Ticker timestamp
}

// GetTickerRequest contains parameters for fetching ticker information
type GetTickerRequest struct {
	Pair string // Trading pair (e.g., "BTC/USD")
}

// GetTickerResponse contains the ticker information
type GetTickerResponse struct {
	Ticker Ticker
}
