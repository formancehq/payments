package binance

import "github.com/formancehq/go-libs/v3/currency"

var (
	// Crypto currencies with their standard decimal precisions
	supportedCurrenciesWithDecimal = map[string]int{
		// Fiat currencies
		"USD": currency.ISO4217Currencies["USD"],
		"EUR": currency.ISO4217Currencies["EUR"],
		"GBP": currency.ISO4217Currencies["GBP"],

		// Major cryptocurrencies
		"BTC":   8,  // Bitcoin - satoshi
		"ETH":   18, // Ethereum - wei
		"BNB":   18, // Binance Coin
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
		"ALGO":  6,  // Algorand
		"ATOM":  6,  // Cosmos
		"FIL":   18, // Filecoin
		"NEAR":  24, // NEAR Protocol

		// Stablecoins
		"USDC":  6,  // USD Coin
		"USDT":  6,  // Tether
		"BUSD":  18, // Binance USD
		"DAI":   18, // Dai
		"TUSD":  18, // TrueUSD
		"USDP":  18, // Pax Dollar
		"FDUSD": 18, // First Digital USD
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
