package modulr

import (
	"fmt"
	"regexp"

	"github.com/formancehq/payments/internal/models"
)

var (
	referencePatternRegexp = regexp.MustCompile("[a-zA-Z0-9 ]*")
)

func (p *Plugin) validateTransferPayoutRequests(pi models.PSPPaymentInitiation) error {
	if pi.SourceAccount == nil {
		return fmt.Errorf("source account is required: %w", models.ErrInvalidRequest)
	}

	if pi.DestinationAccount == nil {
		return fmt.Errorf("destination account is required: %w", models.ErrInvalidRequest)
	}

	if len(pi.Description) > 18 || !referencePatternRegexp.MatchString(pi.Description) {
		return fmt.Errorf("description is invalid: %w", models.ErrInvalidRequest)
	}

	return nil
}
