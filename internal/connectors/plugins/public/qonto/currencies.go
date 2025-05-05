package qonto

import "github.com/formancehq/payments/internal/connectors/plugins/currency"

// Qonto supports only EUR for internal accounts, but external accounts (beneficiaries) can be in other currencies
// Note that the exact currencies supported for external accounts are not documented.

var (
	supportedCurrenciesForInternalAccounts = map[string]int{
		"EUR": currency.ISO4217Currencies["EUR"], //  Euro
	}
	supportedCurrenciesForExternalAccounts = currency.ISO4217Currencies
)
