package stripe

import (
	"fmt"

	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) validatePayoutTransferRequest(pi connector.PSPPaymentInitiation) error {
	if pi.DestinationAccount == nil {
		return connector.NewWrappedError(
			fmt.Errorf("destination account is required in transfer/payout request"),
			connector.ErrInvalidRequest,
		)
	}

	return nil
}
