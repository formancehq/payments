package routable

import "github.com/formancehq/payments/pkg/domain/models"

// Webhooks and bank-account creation are deferred to follow-up PRs;
// MAPPINGS.md §6.4 tracks the roadmap.
var capabilities = []models.Capability{
	models.CAPABILITY_FETCH_ACCOUNTS,
	models.CAPABILITY_FETCH_BALANCES,
	models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
	models.CAPABILITY_FETCH_PAYMENTS,

	models.CAPABILITY_CREATE_TRANSFER,
	models.CAPABILITY_CREATE_PAYOUT,
}
