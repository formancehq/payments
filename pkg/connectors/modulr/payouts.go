package modulr

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/modulr/client"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) createPayout(ctx context.Context, pi connector.PSPPaymentInitiation) (*connector.PSPPayment, error) {
	if err := p.validateTransferPayoutRequests(pi); err != nil {
		return nil, err
	}

	curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return nil, connector.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %v", err),
			connector.ErrInvalidRequest,
		)
	}

	amount, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	if err != nil {
		return nil, connector.NewWrappedError(
			fmt.Errorf("failed to get string amount from big int: %v: %w", pi.Amount, err),
			connector.ErrInvalidRequest,
		)
	}

	description := pi.Description

	resp, err := p.client.InitiatePayout(ctx, &client.PayoutRequest{
		IdempotencyKey:  pi.Reference,
		SourceAccountID: pi.SourceAccount.Reference,
		Destination: client.Destination{
			Type: string(client.DestinationTypeBeneficiary),
			ID:   pi.DestinationAccount.Reference,
		},
		Currency:          curr,
		Amount:            json.Number(amount),
		Reference:         description,
		ExternalReference: description,
	})
	if err != nil {
		return nil, err
	}

	return translatePayoutToPayment(resp)
}

func translatePayoutToPayment(
	from *client.PayoutResponse,
) (*connector.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return nil, err
	}

	status := matchPaymentStatus(from.Status)

	createdAt, err := time.Parse("2006-01-02T15:04:05.999-0700", from.CreatedDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse posted date %s: %w", from.CreatedDate, err)
	}

	precision, ok := supportedCurrenciesWithDecimal[from.Details.Currency]
	if !ok {
		return nil, nil
	}

	amount, err := currency.GetAmountWithPrecisionFromString(from.Details.Amount.String(), precision)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount %s: %w", from.Details.Amount, err)
	}

	return &connector.PSPPayment{
		Reference:                   from.ID,
		CreatedAt:                   createdAt,
		Type:                        connector.PAYMENT_TYPE_PAYOUT,
		Amount:                      amount,
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, from.Details.Currency),
		Scheme:                      connector.PAYMENT_SCHEME_OTHER,
		Status:                      status,
		SourceAccountReference:      &from.Details.SourceAccountID,
		DestinationAccountReference: &from.Details.Destination.ID,
		Raw:                         raw,
	}, nil
}
