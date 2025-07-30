package modulr

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/modulr/client"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferPayoutRequests(pi); err != nil {
		return nil, err
	}

	curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %v", err),
			models.ErrInvalidRequest,
		)
	}

	amount, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get string amount from big int: %v: %w", pi.Amount, err),
			models.ErrInvalidRequest,
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
) (*models.PSPPayment, error) {
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

	return &models.PSPPayment{
		Reference:                   from.ID,
		CreatedAt:                   createdAt,
		Type:                        models.PAYMENT_TYPE_PAYOUT,
		Amount:                      amount,
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, from.Details.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      status,
		SourceAccountReference:      &from.Details.SourceAccountID,
		DestinationAccountReference: &from.Details.Destination.ID,
		Raw:                         raw,
	}, nil
}
