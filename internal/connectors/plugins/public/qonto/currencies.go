package qonto

import "github.com/formancehq/payments/internal/connectors/plugins/currency"

var (
	supportedCurrenciesWithDecimal = map[string]int{
		"EUR": currency.ISO4217Currencies["EUR"], //  Euro
	}
)
