package currencycloud

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/currencycloud/client"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) validatePayoutRequest(pi connector.PSPPaymentInitiation) error {
	if pi.SourceAccount == nil {
		return connector.NewWrappedError(
			fmt.Errorf("source account is required in payout request"),
			connector.ErrInvalidRequest,
		)
	}

	if pi.DestinationAccount == nil {
		return connector.NewWrappedError(
			fmt.Errorf("destination account is required in payout request"),
			connector.ErrInvalidRequest,
		)
	}

	return nil
}

func (p *Plugin) createPayout(ctx context.Context, pi connector.PSPPaymentInitiation) (connector.PSPPayment, error) {
	if err := p.validatePayoutRequest(pi); err != nil {
		return connector.PSPPayment{}, err
	}

	curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return connector.PSPPayment{}, connector.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %v", err),
			connector.ErrInvalidRequest,
		)
	}

	amount, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	if err != nil {
		return connector.PSPPayment{}, connector.NewWrappedError(
			fmt.Errorf("failed to get string amount from big int amount %v: %v", pi.Amount, err),
			connector.ErrInvalidRequest,
		)
	}

	resp, err := p.client.InitiatePayout(ctx, &client.PayoutRequest{
		BeneficiaryID:   pi.DestinationAccount.Reference,
		Currency:        curr,
		Amount:          json.Number(amount),
		Reference:       pi.Description,
		Reason:          pi.Description,
		UniqueRequestID: pi.Reference,
	})
	if err != nil {
		return connector.PSPPayment{}, err
	}

	return translatePayoutToPayment(resp, pi.SourceAccount.Reference)
}

func translatePayoutToPayment(from *client.PayoutResponse, sourceAccountReference string) (connector.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	precision, ok := supportedCurrenciesWithDecimal[from.Currency]
	if !ok {
		return connector.PSPPayment{}, nil
	}

	amount, err := currency.GetAmountWithPrecisionFromString(from.Amount.String(), precision)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	return connector.PSPPayment{
		Reference:                   from.ID,
		CreatedAt:                   from.CreatedAt,
		Type:                        connector.PAYMENT_TYPE_PAYOUT,
		Amount:                      amount,
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, from.Currency),
		Scheme:                      connector.PAYMENT_SCHEME_OTHER,
		Status:                      matchTransactionStatus(from.Status),
		SourceAccountReference:      &sourceAccountReference,
		DestinationAccountReference: &from.BeneficiaryID,
		Raw:                         raw,
	}, nil
}
