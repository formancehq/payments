package coinbaseprime

import "github.com/formancehq/payments/internal/models"

var capabilities = []models.Capability{
	models.CAPABILITY_FETCH_ACCOUNTS,
	models.CAPABILITY_FETCH_BALANCES,
	models.CAPABILITY_FETCH_ORDERS,
	models.CAPABILITY_FETCH_CONVERSIONS,

	models.CAPABILITY_CREATE_ORDER,
	models.CAPABILITY_CANCEL_ORDER,
	models.CAPABILITY_CREATE_CONVERSION,
}
