package coinbaseprime

import "github.com/formancehq/go-libs/v3/currency"

var (
	// Crypto currencies with their standard decimal precisions
	// Most crypto uses 8 decimals (satoshi for BTC, gwei for ETH, etc.)
	supportedCurrenciesWithDecimal = map[string]int{
		// Fiat currencies
		"USD": currency.ISO4217Currencies["USD"],
		"EUR": currency.ISO4217Currencies["EUR"],
		"GBP": currency.ISO4217Currencies["GBP"],

		// Major cryptocurrencies
		"BTC":  8, // Bitcoin - satoshi
		"ETH":  18, // Ethereum - wei (but commonly displayed with 8)
		"LTC":  8, // Litecoin
		"BCH":  8, // Bitcoin Cash
		"XRP":  6, // Ripple
		"XLM":  7, // Stellar
		"LINK": 18, // Chainlink
		"UNI":  18, // Uniswap
		"AAVE": 18, // Aave
		"DOT":  10, // Polkadot
		"SOL":  9, // Solana
		"AVAX": 18, // Avalanche
		"MATIC": 18, // Polygon

		// Stablecoins
		"USDC": 6, // USD Coin
		"USDT": 6, // Tether
		"PYUSD": 6, // PayPal USD
		"DAI":  18, // Dai
		"GUSD": 2, // Gemini Dollar
		"PAX":  18, // Paxos Standard
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
