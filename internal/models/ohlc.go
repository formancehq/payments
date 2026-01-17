package models

import (
	"math/big"
	"time"
)

// OHLCEntry represents a single OHLC (candlestick) data point
type OHLCEntry struct {
	Timestamp time.Time `json:"timestamp"` // Start time of the candle
	Open      *big.Int  `json:"open"`      // Opening price
	High      *big.Int  `json:"high"`      // Highest price
	Low       *big.Int  `json:"low"`       // Lowest price
	Close     *big.Int  `json:"close"`     // Closing price
	Volume    *big.Int  `json:"volume"`    // Trading volume
}

// OHLCData represents OHLC data for a trading pair
type OHLCData struct {
	Pair     string      `json:"pair"`     // Trading pair
	Interval string      `json:"interval"` // Interval (e.g., "1m", "5m", "1h")
	Entries  []OHLCEntry `json:"entries"`  // OHLC data points
}

// GetOHLCRequest contains parameters for fetching OHLC data
type GetOHLCRequest struct {
	Pair     string     // Trading pair (e.g., "BTC/USD")
	Interval string     // Interval: "1m", "5m", "15m", "1h", "4h", "1d"
	Since    *time.Time // Optional: start time for data
	Limit    int        // Optional: limit number of entries
}

// GetOHLCResponse contains the OHLC data
type GetOHLCResponse struct {
	Data OHLCData
}
