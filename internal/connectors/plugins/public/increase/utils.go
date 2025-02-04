package increase

import (
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) validateTransferPayoutRequests(pi models.PSPPaymentInitiation) error {
	if pi.SourceAccount == nil {
		return fmt.Errorf("source account is required: %w", models.ErrInvalidRequest)
	}

	if pi.DestinationAccount == nil {
		return fmt.Errorf("destination account is required: %w", models.ErrInvalidRequest)
	}

	if _, ok := supportedCurrenciesWithDecimal[pi.Asset]; !ok {
		return fmt.Errorf("currency %s is not supported: %w", pi.Asset, models.ErrInvalidRequest)
	}

	return nil
}
