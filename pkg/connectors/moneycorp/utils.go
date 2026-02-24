package moneycorp

import (
	"fmt"

	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) validateTransferPayoutRequests(pi connector.PSPPaymentInitiation) error {
	if pi.SourceAccount == nil {
		return connector.NewWrappedError(
			fmt.Errorf("source account is required in transfer/payout request"),
			connector.ErrInvalidRequest,
		)
	}

	if pi.DestinationAccount == nil {
		return connector.NewWrappedError(
			fmt.Errorf("destination account is required in transfer/payout request"),
			connector.ErrInvalidRequest,
		)
	}

	return nil
}
