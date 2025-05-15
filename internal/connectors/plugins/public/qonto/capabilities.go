package qonto

import "github.com/formancehq/payments/internal/models"

/*
*
Note -- Qonto does have more capabilities, notably webhooks and external transfer creation.
However, to enable them we need to have 3-legged oauth 2 connection, which we don't currently support within Payment.
*/
var capabilities = []models.Capability{
	models.CAPABILITY_FETCH_ACCOUNTS,
	models.CAPABILITY_FETCH_BALANCES,
	models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
	models.CAPABILITY_FETCH_PAYMENTS,
	models.CAPABILITY_CREATE_TRANSFER,
}
