package adyen

import "github.com/formancehq/payments/pkg/connector"

var capabilities = []connector.Capability{
	connector.CAPABILITY_FETCH_ACCOUNTS,
	connector.CAPABILITY_CREATE_WEBHOOKS,
	connector.CAPABILITY_TRANSLATE_WEBHOOKS,
}
