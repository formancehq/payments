package wise

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/pkg/connectors/wise/client"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) createPayout(ctx context.Context, pi connector.PSPPaymentInitiation) (connector.PSPPayment, error) {
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

	resp, err := p.client.CreatePayout(ctx, quote, destinationProfileID, pi.Reference)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	payment, err := fromPayoutToPayment(*resp)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	return payment, nil
}

func fromPayoutToPayment(from client.Payout) (connector.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	precision, ok := supportedCurrenciesWithDecimal[from.TargetCurrency]
	if !ok {
		return connector.PSPPayment{}, connector.NewWrappedError(
			fmt.Errorf("unsupported currency: %s", from.TargetCurrency),
			connector.ErrInvalidRequest,
		)
	}

	amount, err := currency.GetAmountWithPrecisionFromString(from.TargetValue.String(), precision)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	p := connector.PSPPayment{
		Reference: fmt.Sprintf("%d", from.ID),
		CreatedAt: from.CreatedAt,
		Type:      connector.PAYMENT_TYPE_PAYOUT,
		Amount:    amount,
		Asset:     currency.FormatAsset(supportedCurrenciesWithDecimal, from.TargetCurrency),
		Scheme:    connector.PAYMENT_SCHEME_OTHER,
		Status:    matchTransferStatus(from.Status),
		Raw:       raw,
	}

	if from.SourceBalanceID != 0 {
		p.SourceAccountReference = pointer.For(fmt.Sprintf("%d", from.SourceBalanceID))
	}

	if from.TargetAccount != 0 {
		p.DestinationAccountReference = pointer.For(fmt.Sprintf("%d", from.TargetAccount))
	}

	return p, nil
}
