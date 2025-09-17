package coinbaseprime

import "github.com/formancehq/go-libs/v3/currency"

var supportedCurrenciesWithDecimal map[string]int

func init() {
	// Clone ISO 4217 mapping for fiat
	supportedCurrenciesWithDecimal = make(map[string]int, len(currency.ISO4217Currencies)+8)
	for k, v := range currency.ISO4217Currencies {
		supportedCurrenciesWithDecimal[k] = v
	}
	// Add common crypto/stable overrides
	supportedCurrenciesWithDecimal["BTC"] = 8
	supportedCurrenciesWithDecimal["ETH"] = 8
	supportedCurrenciesWithDecimal["SOL"] = 9
	supportedCurrenciesWithDecimal["USDC"] = 6
}
