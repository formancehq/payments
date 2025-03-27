package wise

import (
	"fmt"
	"strconv"

	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func (p *Plugin) validateTransferPayoutRequest(pi models.PSPPaymentInitiation) error {
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

	id, ok := pi.SourceAccount.Metadata["profile_id"]
	if !ok {
		return errorsutils.NewWrappedError(
			fmt.Errorf("source account metadata with profile id is required in transfer/payout request"),
			models.ErrInvalidRequest,
		)
	}

	_, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("source account metadata with profile id is required as an integer in transfer/payout request"),
			models.ErrInvalidRequest,
		)
	}

	id, ok = pi.DestinationAccount.Metadata["profile_id"]
	if !ok {
		return errorsutils.NewWrappedError(
			fmt.Errorf("destination account metadata with profile id is required in transfer/payout request"),
			models.ErrInvalidRequest,
		)
	}

	_, err = strconv.ParseUint(id, 10, 64)
	if err != nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("destination account metadata with profile id is required as an integer in transfer/payout request"),
			models.ErrInvalidRequest,
		)
	}

	return nil
}
