package stripe

import (
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) validatePayoutTransferRequest(pi models.PSPPaymentInitiation) error {
	if pi.DestinationAccount == nil {
		return fmt.Errorf("destination account is required: %w", models.ErrInvalidRequest)
	}

	return nil
}
