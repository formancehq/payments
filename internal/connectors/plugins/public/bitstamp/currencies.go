package bitstamp

import "github.com/formancehq/go-libs/v3/currency"

// supportedCurrenciesWithDecimal maps currency codes to their decimal precision.
// This is essential for correctly converting decimal amounts to integer minor units.
var supportedCurrenciesWithDecimal map[string]int

func init() {
	// Start with standard ISO 4217 fiat currencies (e.g., USD=2, EUR=2, JPY=0)
	supportedCurrenciesWithDecimal = make(map[string]int, len(currency.ISO4217Currencies)+8)
	for k, v := range currency.ISO4217Currencies {
		supportedCurrenciesWithDecimal[k] = v
	}
	// Add cryptocurrency-specific precision (usually 8 for BTC-style, 6 for stablecoins)
	supportedCurrenciesWithDecimal["BTC"] = 8
	supportedCurrenciesWithDecimal["ETH"] = 8
	supportedCurrenciesWithDecimal["SOL"] = 9
	supportedCurrenciesWithDecimal["USDC"] = 6
	supportedCurrenciesWithDecimal["DOGE"] = 8
}
