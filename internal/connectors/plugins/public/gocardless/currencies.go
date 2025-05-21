package gocardless

import "github.com/formancehq/payments/internal/connectors/plugins/currency"

var (
	SupportedCurrenciesWithDecimal = map[string]int{
		"AUD": currency.ISO4217Currencies["AUD"],
		"CAD": currency.ISO4217Currencies["CAD"],
		"DKK": currency.ISO4217Currencies["DKK"],
		"EUR": currency.ISO4217Currencies["EUR"],
		"GBP": currency.ISO4217Currencies["GBP"],
		"NZD": currency.ISO4217Currencies["NZD"],
		"SEK": currency.ISO4217Currencies["SEK"],
		"USD": currency.ISO4217Currencies["USD"],
	}
)
