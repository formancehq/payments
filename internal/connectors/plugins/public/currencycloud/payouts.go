package currencycloud

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/currencycloud/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) validatePayoutRequest(pi models.PSPPaymentInitiation) error {
	if pi.SourceAccount == nil {
		return fmt.Errorf("source account is required: %w", models.ErrInvalidRequest)
	}

	if pi.DestinationAccount == nil {
		return fmt.Errorf("destination account is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) (models.PSPPayment, error) {
	if err := p.validatePayoutRequest(pi); err != nil {
		return models.PSPPayment{}, err
	}

	curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to get currency and precision from asset: %v: %w", err, models.ErrInvalidRequest)
	}

	amount, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to get string amount from big int: %v: %w", err, models.ErrInvalidRequest)
	}

	contact, err := p.client.GetContactID(ctx, pi.SourceAccount.Reference)
	if err != nil {
		return models.PSPPayment{}, err
	}

	resp, err := p.client.InitiatePayout(ctx, &client.PayoutRequest{
		OnBehalfOf:      contact.ID,
		BeneficiaryID:   pi.DestinationAccount.Reference,
		Currency:        curr,
		Amount:          json.Number(amount),
		Reference:       pi.Description,
		UniqueRequestID: pi.Reference,
	})
	if err != nil {
		return models.PSPPayment{}, err
	}

	return translatePayoutToPayment(resp, pi.SourceAccount.Reference)
}

func translatePayoutToPayment(from *client.PayoutResponse, sourceAccountReference string) (models.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return models.PSPPayment{}, err
	}

	precision, ok := supportedCurrenciesWithDecimal[from.Currency]
	if !ok {
		return models.PSPPayment{}, nil
	}

	amount, err := currency.GetAmountWithPrecisionFromString(from.Amount.String(), precision)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return models.PSPPayment{
		Reference:                   from.ID,
		CreatedAt:                   from.CreatedAt,
		Type:                        models.PAYMENT_TYPE_PAYOUT,
		Amount:                      amount,
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, from.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      matchTransactionStatus(from.Status),
		SourceAccountReference:      &sourceAccountReference,
		DestinationAccountReference: &from.BeneficiaryID,
		Raw:                         raw,
	}, nil
}
