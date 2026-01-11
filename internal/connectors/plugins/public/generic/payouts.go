package generic

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/generic/client"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) (models.PSPPayment, error) {
	if err := p.validatePayoutRequest(pi); err != nil {
		return models.PSPPayment{}, err
	}

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
			fmt.Errorf("failed to convert amount to string: %w", err),
			models.ErrInvalidRequest,
		)
	}

	req := &client.PayoutRequest{
		IdempotencyKey:       pi.Reference,
		Amount:               amount,
		Currency:             curr,
		SourceAccountId:      pi.SourceAccount.Reference,
		DestinationAccountId: pi.DestinationAccount.Reference,
		Description:          &pi.Description,
		Metadata:             pi.Metadata,
	}

	resp, err := p.client.CreatePayout(ctx, req)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return payoutResponseToPayment(resp, precision)
}

func (p *Plugin) pollPayoutStatus(ctx context.Context, payoutID string) (models.PSPPayment, error) {
	resp, err := p.client.GetPayoutStatus(ctx, payoutID)
	if err != nil {
		return models.PSPPayment{}, err
	}

	// Look up precision from ISO4217 currency table
	_, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, resp.Currency)
	if err != nil {
		precision = 2 // Default fallback for unknown currencies
	}
	return payoutResponseToPayment(resp, precision)
}

func (p *Plugin) validatePayoutRequest(pi models.PSPPaymentInitiation) error {
	if pi.SourceAccount == nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("source account is required"),
			models.ErrInvalidRequest,
		)
	}

	if pi.DestinationAccount == nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("destination account is required"),
			models.ErrInvalidRequest,
		)
	}

	if pi.Amount == nil || pi.Amount.Cmp(big.NewInt(0)) <= 0 {
		return errorsutils.NewWrappedError(
			fmt.Errorf("amount must be positive"),
			models.ErrInvalidRequest,
		)
	}

	if pi.Reference == "" {
		return errorsutils.NewWrappedError(
			fmt.Errorf("reference is required"),
			models.ErrInvalidRequest,
		)
	}

	return nil
}

func payoutResponseToPayment(resp *client.PayoutResponse, precision int) (models.PSPPayment, error) {
	amount, err := currency.GetAmountWithPrecisionFromString(resp.Amount, precision)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to parse amount %s: %w", resp.Amount, err)
	}

	createdAt, err := time.Parse(time.RFC3339, resp.CreatedAt)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to parse createdAt: %w", err)
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to marshal raw response: %w", err)
	}

	return models.PSPPayment{
		ParentReference:             resp.IdempotencyKey,
		Reference:                   resp.Id,
		CreatedAt:                   createdAt,
		Type:                        models.PAYMENT_TYPE_PAYOUT,
		Amount:                      amount,
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, resp.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      mapStringToPaymentStatus(resp.Status),
		SourceAccountReference:      &resp.SourceAccountId,
		DestinationAccountReference: &resp.DestinationAccountId,
		Metadata:                    resp.Metadata,
		Raw:                         raw,
	}, nil
}

// mapStringToPaymentStatus maps a string status from the external API to the internal PaymentStatus.
// This is used for payout and transfer responses where the status comes as a string.
func mapStringToPaymentStatus(status string) models.PaymentStatus {
	switch status {
	case "SUCCEEDED":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "FAILED":
		return models.PAYMENT_STATUS_FAILED
	case "PENDING":
		return models.PAYMENT_STATUS_PENDING
	default:
		return models.PAYMENT_STATUS_OTHER
	}
}