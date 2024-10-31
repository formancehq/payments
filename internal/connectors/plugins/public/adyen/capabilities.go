package adyen

import "github.com/formancehq/payments/internal/models"

var capabilities = []models.Capability{
	models.CAPABILITY_FETCH_ACCOUNTS,
	models.CAPABILITY_CREATE_WEBHOOKS,
	models.CAPABILITY_TRANSLATE_WEBHOOKS,
}
