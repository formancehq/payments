package kraken

import "github.com/formancehq/go-libs/v3/currency"

var (
	// Crypto currencies with their standard decimal precisions
	supportedCurrenciesWithDecimal = map[string]int{
		// Fiat currencies
		"USD": currency.ISO4217Currencies["USD"],
		"EUR": currency.ISO4217Currencies["EUR"],
		"GBP": currency.ISO4217Currencies["GBP"],
		"CAD": currency.ISO4217Currencies["CAD"],
		"JPY": currency.ISO4217Currencies["JPY"],
		"AUD": currency.ISO4217Currencies["AUD"],
		"CHF": currency.ISO4217Currencies["CHF"],

		// Major cryptocurrencies
		"BTC":   8,  // Bitcoin - satoshi
		"XBT":   8,  // Kraken's Bitcoin symbol
		"ETH":   18, // Ethereum - wei
		"LTC":   8,  // Litecoin
		"BCH":   8,  // Bitcoin Cash
		"XRP":   6,  // Ripple
		"XLM":   7,  // Stellar
		"LINK":  18, // Chainlink
		"UNI":   18, // Uniswap
		"AAVE":  18, // Aave
		"DOT":   10, // Polkadot
		"SOL":   9,  // Solana
		"AVAX":  18, // Avalanche
		"MATIC": 18, // Polygon
		"DOGE":  8,  // Dogecoin
		"ADA":   6,  // Cardano

		// Stablecoins
		"USDC":  6,  // USD Coin
		"USDT":  6,  // Tether
		"DAI":   18, // Dai
	}

	// Default precision for unknown assets
	defaultPrecision = 8
)

// GetPrecision returns the precision for a given asset
func GetPrecision(asset string) int {
	if precision, ok := supportedCurrenciesWithDecimal[asset]; ok {
		return precision
	}
	return defaultPrecision
}
