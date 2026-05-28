package atlar

import "github.com/formancehq/go-libs/v5/pkg/types/currency"

var (
	supportedCurrenciesWithDecimal = map[string]int{
		"EUR": currency.ISO4217Currencies["EUR"], //  Euro
		"DKK": currency.ISO4217Currencies["DKK"],
	}
)
