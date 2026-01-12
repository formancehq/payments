package fireblocks

import "github.com/formancehq/go-libs/v3/currency"

var (
	// Supported currencies with their decimal precisions
	supportedCurrenciesWithDecimal = map[string]int{
		// Fiat currencies from ISO 4217
		"USD": currency.ISO4217Currencies["USD"],
		"EUR": currency.ISO4217Currencies["EUR"],
		"GBP": currency.ISO4217Currencies["GBP"],
		"CHF": currency.ISO4217Currencies["CHF"],
		"JPY": currency.ISO4217Currencies["JPY"],
		"CAD": currency.ISO4217Currencies["CAD"],
		"AUD": currency.ISO4217Currencies["AUD"],

		// Major cryptocurrencies
		"BTC":       8,
		"BTC_TEST":  8,
		"ETH":       18,
		"ETH_TEST":  18,
		"ETH_TEST3": 18,
		"ETH_TEST5": 18,
		"ETH_TEST6": 18,
		"SOL":       9,
		"SOL_TEST":  9,
		"XRP":       6,
		"XRP_TEST":  6,
		"LTC":       8,
		"LTC_TEST":  8,
		"BCH":       8,
		"BCH_TEST":  8,
		"XLM":       7,
		"XLM_TEST":  7,
		"DOGE":      8,
		"DOGE_TEST": 8,
		"DOT":       10,
		"ADA":       6,
		"AVAX":      18,
		"AVAXC":     18,
		"MATIC":     18,
		"MATIC_POLYGON": 18,
		"ATOM":      6,
		"ALGO":      6,
		"NEAR":      24,
		"FTM":       18,
		"ONE":       18,
		"CELO":      18,
		"FLOW":      8,

		// Stablecoins
		"USDC":       6,
		"USDC_E":     6,
		"USDC_ETH":   6,
		"USDC_SOL":   6,
		"USDT":       6,
		"USDT_ERC20": 6,
		"USDT_TRC20": 6,
		"USDT_SOL":   6,
		"DAI":        18,
		"BUSD":       18,
		"TUSD":       18,
		"USDP":       18,
		"FRAX":       18,
		"LUSD":       18,
		"GUSD":       2,
		"PYUSD":      6,

		// Wrapped tokens
		"WBTC":   8,
		"WETH":   18,
		"STETH":  18,
		"WSTETH": 18,
		"RETH":   18,
		"CBETH":  18,

		// DeFi tokens
		"UNI":   18,
		"AAVE":  18,
		"LINK":  18,
		"MKR":   18,
		"CRV":   18,
		"COMP":  18,
		"SUSHI": 18,
		"SNX":   18,
		"YFI":   18,
		"BAL":   18,
		"1INCH": 18,

		// Exchange tokens
		"BNB": 18,
		"FTT": 18,
		"CRO": 8,
		"LEO": 18,
		"OKB": 18,
		"HT":  18,
		"KCS": 6,

		// Layer 2 tokens
		"ARB": 18,
		"OP":  18,
		"IMX": 18,

		// Gaming/Metaverse
		"MANA": 18,
		"SAND": 18,
		"AXS":  18,
		"ENJ":  18,
		"GALA": 8,
		"APE":  18,
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
