package krakenpro

import "github.com/formancehq/payments/pkg/domain/models"

// Read-only crypto-exchange capability set per EN-1014.
// Transfers / payouts / bank accounts / webhooks are intentionally
// omitted — this connector is read-only by design.
var capabilities = []models.Capability{
	models.CAPABILITY_FETCH_ACCOUNTS,
	models.CAPABILITY_FETCH_BALANCES,
	models.CAPABILITY_FETCH_PAYMENTS,
	models.CAPABILITY_FETCH_ORDERS,
	models.CAPABILITY_FETCH_CONVERSIONS,
}
