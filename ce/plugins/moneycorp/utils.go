package moneycorp

import (
	"fmt"

	"github.com/formancehq/payments/pkg/domain/models"
	errorsutils "github.com/formancehq/payments/pkg/domain/errors"
)

func (p *Plugin) validateTransferPayoutRequests(pi models.PSPPaymentInitiation) error {
	if pi.SourceAccount == nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("source account is required in transfer/payout request"),
			models.ErrInvalidRequest,
		)
	}

	if pi.DestinationAccount == nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("destination account is required in transfer/payout request"),
			models.ErrInvalidRequest,
		)
	}

	return nil
}
