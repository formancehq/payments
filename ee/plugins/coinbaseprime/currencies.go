package coinbaseprime

import "github.com/formancehq/go-libs/v3/currency"

// fiatCurrenciesFallback provides ISO4217 fiat currency precisions as a
// fallback. Crypto asset precisions are loaded dynamically from the
// Coinbase Prime API at install time.
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
