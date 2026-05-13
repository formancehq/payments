package universal

import "github.com/formancehq/payments/internal/models"

// capabilities is the static superset advertised at registration time.
//
// The Universal Connector registers every capability the engine knows about so
// the catalog reflects everything a counterparty *could* implement. The set the
// counterparty actually exposes is discovered at install time via
// GET /v1/capabilities; runtime guards in guard.go enforce the per-install
// subset and return plugins.ErrNotImplemented for anything not declared.
var capabilities = []models.Capability{
	models.CAPABILITY_FETCH_ACCOUNTS,
	models.CAPABILITY_FETCH_BALANCES,
	models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
	models.CAPABILITY_FETCH_PAYMENTS,
	models.CAPABILITY_FETCH_OTHERS,
	models.CAPABILITY_FETCH_ORDERS,
	models.CAPABILITY_FETCH_CONVERSIONS,

	models.CAPABILITY_CREATE_WEBHOOKS,
	models.CAPABILITY_TRANSLATE_WEBHOOKS,

	models.CAPABILITY_CREATE_BANK_ACCOUNT,
	models.CAPABILITY_CREATE_TRANSFER,
	models.CAPABILITY_CREATE_PAYOUT,

	models.CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION,
	models.CAPABILITY_ALLOW_FORMANCE_PAYMENT_CREATION,
}
