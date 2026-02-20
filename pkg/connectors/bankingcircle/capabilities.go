package bankingcircle

import "github.com/formancehq/payments/pkg/connector"

var capabilities = []connector.Capability{
	connector.CAPABILITY_FETCH_ACCOUNTS,
	connector.CAPABILITY_FETCH_PAYMENTS,
	connector.CAPABILITY_FETCH_BALANCES,

	connector.CAPABILITY_CREATE_BANK_ACCOUNT,
	connector.CAPABILITY_CREATE_TRANSFER,
	connector.CAPABILITY_CREATE_PAYOUT,
}
