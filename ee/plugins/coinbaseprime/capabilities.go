package coinbaseprime

import "github.com/formancehq/payments/pkg/domain/models"

var capabilities = []models.Capability{
	models.CAPABILITY_FETCH_ACCOUNTS,
	models.CAPABILITY_FETCH_BALANCES,
	models.CAPABILITY_FETCH_PAYMENTS,
	models.CAPABILITY_FETCH_ORDERS,
	models.CAPABILITY_FETCH_CONVERSIONS,
}
