package krakenpro

import "github.com/formancehq/go-libs/v3/currency"

// fiatCurrenciesFallback provides ISO4217 fiat currency precisions.
var fiatCurrenciesFallback = map[string]int{
	"USD": currency.ISO4217Currencies["USD"],
	"EUR": currency.ISO4217Currencies["EUR"],
	"GBP": currency.ISO4217Currencies["GBP"],
	"CAD": currency.ISO4217Currencies["CAD"],
	"AUD": currency.ISO4217Currencies["AUD"],
	"JPY": currency.ISO4217Currencies["JPY"],
	"CHF": currency.ISO4217Currencies["CHF"],
	"SGD": currency.ISO4217Currencies["SGD"],
}

// cryptoCurrenciesPrecision provides well-known crypto asset precisions.
// Kraken does not expose precision via API, so we use a static map.
var cryptoCurrenciesPrecision = map[string]int{
	"BTC":  8,
	"ETH":  18,
	"XRP":  6,
	"LTC":  8,
	"BCH":  8,
	"ADA":  6,
	"DOT":  10,
	"LINK": 18,
	"XLM":  7,
	"DOGE": 8,
	"SOL":  9,
	"AVAX": 18,
	"MATIC": 18,
	"ATOM": 6,
	"UNI":  18,
	"USDT": 6,
	"USDC": 6,
	"DAI":  18,
}

// defaultPrecision is used for assets not in the known maps.
const defaultPrecision = 8
