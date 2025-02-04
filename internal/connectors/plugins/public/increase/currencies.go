package increase

import "github.com/formancehq/payments/internal/connectors/plugins/currency"

var (
	// TODO: the next line tells that the connector is supporting all currencies.
	// If you only want to support specific currencies, you will have to remove
	// this line and set the map yourselves
	// Example:
	// supportedCurrenciesWithDecimal = map[string]int{
	// 	"EUR": currency.ISO4217Currencies["EUR"], //  Euro
	// 	"DKK": currency.ISO4217Currencies["DKK"],
	// }
	supportedCurrenciesWithDecimal = currency.ISO4217Currencies
)
