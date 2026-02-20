package moneycorp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/moneycorp/client"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) createPayout(ctx context.Context, pi connector.PSPPaymentInitiation) (*connector.PSPPayment, error) {
	if err := p.validateTransferPayoutRequests(pi); err != nil {
		return nil, err
	}

	curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return nil, connector.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %w", err),
			connector.ErrInvalidRequest,
		)
	}

	amount, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	if err != nil {
		return nil, connector.NewWrappedError(
			fmt.Errorf("failed to get string amount from big int %v, %w", pi.Amount, err),
			connector.ErrInvalidRequest,
		)
	}

	resp, err := p.client.InitiatePayout(
		ctx,
		&client.PayoutRequest{
			IdempotencyKey:   pi.Reference,
			SourceAccountID:  pi.SourceAccount.Reference,
			RecipientID:      pi.DestinationAccount.Reference,
			PaymentAmount:    json.Number(amount),
			PaymentCurrency:  curr,
			PaymentMethod:    "Standard",
			PaymentReference: pi.Description,
			ClientReference:  pi.Description,
		},
	)
	if err != nil {
		return nil, err
	}

	return payoutToPayment(resp)
}

func payoutToPayment(from *client.PayoutResponse) (*connector.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction: %w", err)
	}

	createdAt, err := time.Parse("2006-01-02T15:04:05.999999999", from.Attributes.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction date: %w", err)
	}

	c, err := currency.GetPrecision(supportedCurrenciesWithDecimal, from.Attributes.PaymentCurrency)
	if err != nil {
		return nil, err
	}

	amount, err := currency.GetAmountWithPrecisionFromString(from.Attributes.PaymentAmount.String(), c)
	if err != nil {
		return nil, err
	}

	return &connector.PSPPayment{
		Reference:              from.ID,
		CreatedAt:              createdAt,
		Type:                   connector.PAYMENT_TYPE_PAYOUT,
		Amount:                 amount,
		Asset:                  currency.FormatAsset(supportedCurrenciesWithDecimal, from.Attributes.PaymentCurrency),
		Scheme:                 connector.PAYMENT_SCHEME_OTHER,
		Status:                 matchPaymentStatus(from.Attributes.PaymentStatus),
		SourceAccountReference: pointer.For(fmt.Sprintf("%d", from.Attributes.AccountID)),
		DestinationAccountReference: func() *string {
			if from.Attributes.RecipientDetails.RecipientID == 0 {
				return nil
			}
			return pointer.For(fmt.Sprintf("%d", from.Attributes.RecipientDetails.RecipientID))
		}(),
		Raw: raw,
	}, nil
}

func matchPaymentStatus(status string) connector.PaymentStatus {
	// Unauthorised, Awaiting Dispatch, Sent, Cleared, Failed, Cancelled or Query
	switch status {
	case "Unauthorised", "Failed", "Query":
		return connector.PAYMENT_STATUS_FAILED
	case "Awaiting Dispatch", "Sent":
		return connector.PAYMENT_STATUS_PENDING
	case "Cleared":
		return connector.PAYMENT_STATUS_SUCCEEDED
	case "Cancelled":
		return connector.PAYMENT_STATUS_FAILED
	}

	return connector.PAYMENT_STATUS_UNKNOWN
}
