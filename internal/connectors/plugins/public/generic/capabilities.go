package generic

import "github.com/formancehq/payments/internal/models"

var capabilities = []models.Capability{
	models.CAPABILITY_FETCH_ACCOUNTS,
	models.CAPABILITY_FETCH_BALANCES,
	models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
	models.CAPABILITY_FETCH_PAYMENTS,

	models.CAPABILITY_CREATE_PAYOUT,

	models.CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION,
	models.CAPABILITY_ALLOW_FORMANCE_PAYMENT_CREATION,
}
