package wise

import (
	"fmt"
	"strconv"

	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) validateTransferPayoutRequest(pi connector.PSPPaymentInitiation) error {
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

	id, ok := pi.SourceAccount.Metadata["profile_id"]
	if !ok {
		return connector.NewWrappedError(
			fmt.Errorf("source account metadata with profile id is required in transfer/payout request"),
			connector.ErrInvalidRequest,
		)
	}

	_, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return connector.NewWrappedError(
			fmt.Errorf("source account metadata with profile id is required as an integer in transfer/payout request"),
			connector.ErrInvalidRequest,
		)
	}

	id, ok = pi.DestinationAccount.Metadata["profile_id"]
	if !ok {
		return connector.NewWrappedError(
			fmt.Errorf("destination account metadata with profile id is required in transfer/payout request"),
			connector.ErrInvalidRequest,
		)
	}

	_, err = strconv.ParseUint(id, 10, 64)
	if err != nil {
		return connector.NewWrappedError(
			fmt.Errorf("destination account metadata with profile id is required as an integer in transfer/payout request"),
			connector.ErrInvalidRequest,
		)
	}

	return nil
}
