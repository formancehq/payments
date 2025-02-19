package gocardless

import "github.com/formancehq/payments/internal/models"

var Capabilities = []models.Capability{
	models.CAPABILITY_FETCH_OTHERS,
	models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
	models.CAPABILITY_FETCH_PAYMENTS,

	models.CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION,
}
