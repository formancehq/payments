package column

import "github.com/formancehq/payments/internal/connectors/plugins/currency"

var (
	supportedCurrenciesWithDecimal = map[string]int{
		"USD": currency.ISO4217Currencies["USD"], // US Dollar
	}
)
