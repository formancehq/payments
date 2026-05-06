package routable

import "github.com/formancehq/payments/internal/models"

// capabilities matches the feature set exposed by the standalone
// connector-routable service today (fetch + create payouts/transfers via
// Routable payables) so this PR is feature-parity with Generic-Connector
// Routable. Webhooks and bank-account creation are deferred to follow-up PRs.
var capabilities = []models.Capability{
	models.CAPABILITY_FETCH_ACCOUNTS,
	models.CAPABILITY_FETCH_BALANCES,
	models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
	models.CAPABILITY_FETCH_PAYMENTS,

	models.CAPABILITY_CREATE_TRANSFER,
	models.CAPABILITY_CREATE_PAYOUT,
}
