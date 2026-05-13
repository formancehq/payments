package teller

import "github.com/formancehq/payments/internal/models"

var capabilities = []models.Capability{
	models.CAPABILITY_FETCH_ACCOUNTS,
	models.CAPABILITY_FETCH_BALANCES,
	models.CAPABILITY_FETCH_PAYMENTS,
}
