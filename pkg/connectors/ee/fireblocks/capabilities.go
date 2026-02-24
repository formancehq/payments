package fireblocks

import "github.com/formancehq/payments/pkg/connector"

var capabilities = []connector.Capability{
	connector.CAPABILITY_FETCH_ACCOUNTS,
	connector.CAPABILITY_FETCH_BALANCES,
	connector.CAPABILITY_FETCH_PAYMENTS,
}
