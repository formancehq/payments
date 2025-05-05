package moov

import (
	"fmt"

	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func (p *Plugin) validateTransferPayoutRequest(pi models.PSPPaymentInitiation) error {
	if pi.Amount == nil {
		return models.NewConnectorValidationError("Amount", fmt.Errorf("amount is required"))
	}

	if pi.Asset == "" {
		return models.NewConnectorValidationError("Asset", fmt.Errorf("asset is required"))
	}

	// Validate currency
	curr, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return models.NewConnectorValidationError("Asset", errorsutils.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %w", err),
			models.ErrInvalidRequest,
		))
	}

	// Check if currency is supported
	if _, ok := supportedCurrenciesWithDecimal[curr]; !ok {
		return models.NewConnectorValidationError("Asset", fmt.Errorf("unsupported currency: %s", curr))
	}

	return nil
}