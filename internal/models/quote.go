package models

import (
	"math/big"
	"time"
)

// Quote represents a price quote for a trade
type Quote struct {
	SourceAsset   string    `json:"sourceAsset"`
	TargetAsset   string    `json:"targetAsset"`
	Direction     string    `json:"direction"` // BUY or SELL
	Quantity      *big.Int  `json:"quantity"`
	Price         *big.Int  `json:"price"`         // Estimated price per unit
	TotalPrice    *big.Int  `json:"totalPrice"`    // Total price for the quantity
	Fee           *big.Int  `json:"fee,omitempty"` // Estimated fee
	ExpiresAt     time.Time `json:"expiresAt"`     // When the quote expires
	Timestamp     time.Time `json:"timestamp"`     // When the quote was generated
}

// GetQuoteRequest contains parameters for requesting a quote
type GetQuoteRequest struct {
	SourceAsset string   // Source asset (e.g., "BTC")
	TargetAsset string   // Target asset (e.g., "USD")
	Direction   string   // BUY or SELL
	Quantity    *big.Int // Quantity to trade
}

// GetQuoteResponse contains the quote data
type GetQuoteResponse struct {
	Quote Quote
}
