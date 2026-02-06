package wise

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) createTransfer(ctx context.Context, pi connector.PSPPaymentInitiation) (connector.PSPPayment, error) {
	if err := p.validateTransferPayoutRequest(pi); err != nil {
		return connector.PSPPayment{}, err
	}

	sourceProfileID := pi.SourceAccount.Metadata["profile_id"]
	destinationProfileID, _ := strconv.ParseUint(pi.DestinationAccount.Metadata["profile_id"], 10, 64)

	curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return connector.PSPPayment{}, connector.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %w", err),
			connector.ErrInvalidRequest,
		)
	}

	amount, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	if err != nil {
		return connector.PSPPayment{}, connector.NewWrappedError(
			fmt.Errorf("failed to convert big int amount to string %v: %w", pi.Amount, err),
			connector.ErrInvalidRequest,
		)
	}

	quote, err := p.client.CreateQuote(ctx, sourceProfileID, curr, json.Number(amount))
	if err != nil {
		return connector.PSPPayment{}, err
	}

	resp, err := p.client.CreateTransfer(ctx, quote, destinationProfileID, pi.Reference)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	payment, err := fromTransferToPayment(*resp)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	return payment, nil
}
