package bitstamp

import "github.com/formancehq/payments/internal/models"

// capabilities defines what operations this Bitstamp connector supports:
// - FETCH_ACCOUNTS: Retrieve configured Bitstamp sub-accounts
// - FETCH_BALANCES: Get current balances for each account (fiat + crypto)
// - FETCH_PAYMENTS: Poll historical transactions from user_transactions API
var capabilities = []models.Capability{
	models.CAPABILITY_FETCH_ACCOUNTS,
	models.CAPABILITY_FETCH_BALANCES,
	models.CAPABILITY_FETCH_PAYMENTS,
	models.CAPABILITY_FETCH_TRADES,
}
