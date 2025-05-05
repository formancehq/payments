package moov

import "github.com/formancehq/payments/internal/connectors/plugins/currency"

var (
	// Moov supports USD only for now
	supportedCurrenciesWithDecimal = map[string]int{
		"USD": 2,
	}
)