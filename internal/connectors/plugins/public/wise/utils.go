package wise

import (
	"fmt"
	"strconv"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) validateTransferPayoutRequest(pi models.PSPPaymentInitiation) error {
	if pi.SourceAccount == nil {
		return fmt.Errorf("source account is required: %w", models.ErrInvalidRequest)
	}

	if pi.DestinationAccount == nil {
		return fmt.Errorf("destination account is required: %w", models.ErrInvalidRequest)
	}

	id, ok := pi.SourceAccount.Metadata["profile_id"]
	if !ok {
		return fmt.Errorf("source account metadata with profile id is required: %w", models.ErrInvalidRequest)
	}

	_, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return fmt.Errorf("source account metadata with profile id is required as an integer: %w", models.ErrInvalidRequest)
	}

	id, ok = pi.DestinationAccount.Metadata["profile_id"]
	if !ok {
		return fmt.Errorf("destination account metadata with profile id is required: %w", models.ErrInvalidRequest)
	}

	_, err = strconv.ParseUint(id, 10, 64)
	if err != nil {
		return fmt.Errorf("destination account metadata with profile id is required as an integer: %w", models.ErrInvalidRequest)
	}

	return nil
}
