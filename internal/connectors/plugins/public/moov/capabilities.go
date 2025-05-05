package moov

import "github.com/formancehq/payments/internal/models"

var capabilities = []models.Capability{
	models.CAPABILITY_FETCH_ACCOUNTS,
	models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
	models.CAPABILITY_FETCH_PAYMENTS,
	models.CAPABILITY_FETCH_OTHERS,
	models.CAPABILITY_CREATE_TRANSFER,
	models.CAPABILITY_CREATE_PAYOUT,
}