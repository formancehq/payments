package column

import "github.com/formancehq/go-libs/v3/currency"

var (
	supportedCurrenciesWithDecimal = map[string]int{
		"USD": currency.ISO4217Currencies["USD"], // US Dollar
	}
)
