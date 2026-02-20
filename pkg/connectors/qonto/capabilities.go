package qonto

import "github.com/formancehq/payments/pkg/connector"

/*
*
Note -- Qonto does have more capabilities, notably webhooks and external transfer creation.
However, to enable them we need to have 3-legged oauth 2 connection, which we don't currently support within Payment.
*/
var capabilities = []connector.Capability{
	connector.CAPABILITY_FETCH_ACCOUNTS,
	connector.CAPABILITY_FETCH_BALANCES,
	connector.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
	connector.CAPABILITY_FETCH_PAYMENTS,
}
