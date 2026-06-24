package atlar

import "github.com/formancehq/payments/pkg/domain/models"

var capabilities = []models.Capability{
	models.CAPABILITY_FETCH_ACCOUNTS,
	models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
	models.CAPABILITY_FETCH_PAYMENTS,
	models.CAPABILITY_FETCH_OTHERS,
}
