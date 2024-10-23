package wise

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (models.PSPPayment, error) {
	if err := p.validateTransferPayoutRequest(pi); err != nil {
		return models.PSPPayment{}, err
	}

	sourceProfileID := pi.SourceAccount.Metadata["profile_id"]
	destinationProfileID, _ := strconv.ParseUint(pi.DestinationAccount.Metadata["profile_id"], 10, 64)

	curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to get currency and precision from asset: %w: %w", err, models.ErrInvalidRequest)
	}

	amount, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to convert amount to string: %w: %w", err, models.ErrInvalidRequest)
	}

	quote, err := p.client.CreateQuote(ctx, sourceProfileID, curr, json.Number(amount))
	if err != nil {
		return models.PSPPayment{}, err
	}

	resp, err := p.client.CreateTransfer(ctx, quote, destinationProfileID, pi.Reference)
	if err != nil {
		return models.PSPPayment{}, err
	}

	payment, err := fromTransferToPayment(*resp)
	if err != nil {
		return models.PSPPayment{}, err
	}

	if payment == nil {
		return models.PSPPayment{}, fmt.Errorf("unsupported payments: %w", models.ErrInvalidRequest)
	}

	return *payment, nil
}
