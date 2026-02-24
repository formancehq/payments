package atlar

import "github.com/formancehq/payments/pkg/connector"

var capabilities = []connector.Capability{
	connector.CAPABILITY_FETCH_ACCOUNTS,
	connector.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
	connector.CAPABILITY_FETCH_PAYMENTS,
	connector.CAPABILITY_FETCH_OTHERS,
}
