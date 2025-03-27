package modulr

import (
	"fmt"
	"regexp"

	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

var (
	referencePatternRegexp = regexp.MustCompile("[a-zA-Z0-9 ]*")
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

	if len(pi.Description) > 18 || !referencePatternRegexp.MatchString(pi.Description) {
		return errorsutils.NewWrappedError(
			fmt.Errorf("description must be less than 18 characters and match the following regexp [a-zA-Z0-9 ]*: %s", pi.Description),
			models.ErrInvalidRequest,
		)
	}

	return nil
}
