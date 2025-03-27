package wise

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/wise/client"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) (models.PSPPayment, error) {
	if err := p.validateTransferPayoutRequest(pi); err != nil {
		return models.PSPPayment{}, err
	}

	sourceProfileID := pi.SourceAccount.Metadata["profile_id"]
	destinationProfileID, _ := strconv.ParseUint(pi.DestinationAccount.Metadata["profile_id"], 10, 64)

	curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return models.PSPPayment{}, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %w", err),
			models.ErrInvalidRequest,
		)
	}

	amount, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	if err != nil {
		return models.PSPPayment{}, errorsutils.NewWrappedError(
			fmt.Errorf("failed to convert big int amount to string %v: %w", pi.Amount, err),
			models.ErrInvalidRequest,
		)
	}

	quote, err := p.client.CreateQuote(ctx, sourceProfileID, curr, json.Number(amount))
	if err != nil {
		return models.PSPPayment{}, err
	}

	resp, err := p.client.CreatePayout(ctx, quote, destinationProfileID, pi.Reference)
	if err != nil {
		return models.PSPPayment{}, err
	}

	payment, err := fromPayoutToPayment(*resp)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return payment, nil
}

func fromPayoutToPayment(from client.Payout) (models.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return models.PSPPayment{}, err
	}

	precision, ok := supportedCurrenciesWithDecimal[from.TargetCurrency]
	if !ok {
		return models.PSPPayment{}, errorsutils.NewWrappedError(
			fmt.Errorf("unsupported currency: %s", from.TargetCurrency),
			models.ErrInvalidRequest,
		)
	}

	amount, err := currency.GetAmountWithPrecisionFromString(from.TargetValue.String(), precision)
	if err != nil {
		return models.PSPPayment{}, err
	}

	p := models.PSPPayment{
		Reference: fmt.Sprintf("%d", from.ID),
		CreatedAt: from.CreatedAt,
		Type:      models.PAYMENT_TYPE_PAYOUT,
		Amount:    amount,
		Asset:     currency.FormatAsset(supportedCurrenciesWithDecimal, from.TargetCurrency),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
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
