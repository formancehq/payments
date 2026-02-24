package stripe

import "github.com/formancehq/payments/pkg/connector"

var capabilities = []connector.Capability{
	connector.CAPABILITY_FETCH_ACCOUNTS,
	connector.CAPABILITY_FETCH_BALANCES,
	connector.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
	connector.CAPABILITY_FETCH_PAYMENTS,

	connector.CAPABILITY_CREATE_TRANSFER,
	connector.CAPABILITY_CREATE_PAYOUT,

	connector.CAPABILITY_CREATE_WEBHOOKS,
	connector.CAPABILITY_TRANSLATE_WEBHOOKS,
}
