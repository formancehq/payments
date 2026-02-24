package modulr

import (
	"fmt"
	"regexp"

	"github.com/formancehq/payments/pkg/connector"
)

var (
	referencePatternRegexp = regexp.MustCompile("[a-zA-Z0-9 ]*")
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

	if len(pi.Description) > 18 || !referencePatternRegexp.MatchString(pi.Description) {
		return connector.NewWrappedError(
			fmt.Errorf("description must be less than 18 characters and match the following regexp [a-zA-Z0-9 ]*: %s", pi.Description),
			connector.ErrInvalidRequest,
		)
	}

	return nil
}
