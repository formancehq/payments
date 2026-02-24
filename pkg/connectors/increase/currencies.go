package increase

import "github.com/formancehq/go-libs/v3/currency"

var (
	supportedCurrenciesWithDecimal = map[string]int{
		"CAD": currency.ISO4217Currencies["CAD"], // Canadian Dollar
		"CHF": currency.ISO4217Currencies["CHF"], // Swiss Franc
		"EUR": currency.ISO4217Currencies["EUR"], // Euro
		"GBP": currency.ISO4217Currencies["GBP"], // British Pound
		"JPY": currency.ISO4217Currencies["JPY"], // Japanese Yen
		"USD": currency.ISO4217Currencies["USD"], // US Dollar
	}
)
