package models

// TradableAsset represents a trading pair available on an exchange
type TradableAsset struct {
	Pair           string `json:"pair"`           // Trading pair (e.g., "BTC/USD")
	BaseAsset      string `json:"baseAsset"`      // Base asset (e.g., "BTC")
	QuoteAsset     string `json:"quoteAsset"`     // Quote asset (e.g., "USD")
	MinOrderSize   string `json:"minOrderSize"`   // Minimum order size
	MaxOrderSize   string `json:"maxOrderSize"`   // Maximum order size (empty if unlimited)
	PricePrecision int    `json:"pricePrecision"` // Number of decimal places for price
	SizePrecision  int    `json:"sizePrecision"`  // Number of decimal places for size
	Status         string `json:"status"`         // Status (e.g., "online", "offline")
}

// GetTradableAssetsRequest contains parameters for fetching tradable assets
type GetTradableAssetsRequest struct {
	// Optional filter for specific pairs
	Pairs []string
}

// GetTradableAssetsResponse contains the list of tradable assets
type GetTradableAssetsResponse struct {
	Assets []TradableAsset
}
