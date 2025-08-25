package checkout

import "github.com/formancehq/payments/internal/models"

var capabilities = []models.Capability{
	// TODO: add or remove capabilities depending on what data your connector
	// intends to import
	models.CAPABILITY_FETCH_ACCOUNTS, // OK
	models.CAPABILITY_FETCH_BALANCES, // OK
	models.CAPABILITY_FETCH_PAYMENTS, // OK

	models.CAPABILITY_CREATE_TRANSFER,
	models.CAPABILITY_CREATE_PAYOUT,
}
