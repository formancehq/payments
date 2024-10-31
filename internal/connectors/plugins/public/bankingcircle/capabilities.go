package bankingcircle

import "github.com/formancehq/payments/internal/models"

var capabilities = []models.Capability{
	models.CAPABILITY_FETCH_ACCOUNTS,
	models.CAPABILITY_FETCH_PAYMENTS,
	models.CAPABILITY_FETCH_BALANCES,

	models.CAPABILITY_CREATE_BANK_ACCOUNT,
	models.CAPABILITY_CREATE_TRANSFER,
	models.CAPABILITY_CREATE_PAYOUT,
}
