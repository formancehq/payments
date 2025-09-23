package stripe

import (
    "fmt"

    "github.com/formancehq/payments/internal/models"
    errorsutils "github.com/formancehq/payments/pkg/pluginsdk/errors"
)

func (p *Plugin) validatePayoutTransferRequest(pi models.PSPPaymentInitiation) error {
	if pi.DestinationAccount == nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("destination account is required in transfer/payout request"),
			models.ErrInvalidRequest,
		)
	}

	return nil
}
