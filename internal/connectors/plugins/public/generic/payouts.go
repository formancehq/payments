package generic

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
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

	amount := amountToString(*pi.Amount, precision)

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

	// Get precision from currency (assuming USD for now, this should be from the original request)
	precision := int(2) // Default for USD
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
	// Parse amount - handle both integer and decimal formats
	amount, err := parseAmountFromString(resp.Amount, precision)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to parse amount: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, resp.CreatedAt)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to parse created at: %w", err)
	}

	status := models.PAYMENT_STATUS_PENDING
	switch resp.Status {
	case "SUCCEEDED":
		status = models.PAYMENT_STATUS_SUCCEEDED
	case "FAILED":
		status = models.PAYMENT_STATUS_FAILED
	case "PENDING":
		status = models.PAYMENT_STATUS_PENDING
	}

	// Create raw JSON for the payment
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
		Asset:                       resp.Currency + "/2", // Assuming 2 decimal precision
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      status,
		SourceAccountReference:      &resp.SourceAccountId,
		DestinationAccountReference: &resp.DestinationAccountId,
		Metadata:                    resp.Metadata,
		Raw:                         raw,
	}, nil
}

func amountToString(amount big.Int, precision int) string {
	raw := amount.String()
	if precision < 0 {
		precision = 0
	}
	insertPosition := len(raw) - precision
	if insertPosition <= 0 {
		return "0." + strings.Repeat("0", -insertPosition) + raw
	}
	return raw[:insertPosition] + "." + raw[insertPosition:]
}

func parseAmountFromString(amountStr string, precision int) (*big.Int, error) {
	if precision < 0 {
		precision = 0
	}

	// If it contains a decimal point, handle it
	if strings.Contains(amountStr, ".") {
		parts := strings.Split(amountStr, ".")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid decimal format: %s", amountStr)
		}

		integerPart := parts[0]
		decimalPart := parts[1]

		// Pad or truncate decimal part to match precision
		if len(decimalPart) > precision {
			decimalPart = decimalPart[:precision]
		} else if len(decimalPart) < precision {
			decimalPart = decimalPart + strings.Repeat("0", precision-len(decimalPart))
		}

		// Combine integer and decimal parts
		combinedStr := integerPart + decimalPart
		amount, ok := new(big.Int).SetString(combinedStr, 10)
		if !ok {
			return nil, fmt.Errorf("failed to parse combined amount: %s", combinedStr)
		}
		return amount, nil
	}

	// If no decimal point, assume it's already in minor units
	amount, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse integer amount: %s", amountStr)
	}
	return amount, nil
}