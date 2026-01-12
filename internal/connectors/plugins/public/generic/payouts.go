package generic

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/generic/client"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) (models.PSPPayment, error) {
	if err := p.validatePayoutRequest(pi); err != nil {
		return models.PSPPayment{}, err
	}

	// Validate asset format (e.g., "USD/2", "BTC/8", "COIN", "JPY")
	if _, _, err := parseAssetUMN(pi.Asset); err != nil {
		return models.PSPPayment{}, errorsutils.NewWrappedError(
			fmt.Errorf("failed to parse asset %s: %w", pi.Asset, err),
			models.ErrInvalidRequest,
		)
	}

	// All amounts are integers (minor units) - pass through directly
	req := &client.PayoutRequest{
		IdempotencyKey:       pi.Reference,
		Amount:               pi.Amount.String(),
		Currency:             pi.Asset, // Pass full UMN in currency field: "USD/2", "BTC/8"
		SourceAccountId:      pi.SourceAccount.Reference,
		DestinationAccountId: pi.DestinationAccount.Reference,
		Description:          &pi.Description,
		Metadata:             pi.Metadata,
	}

	resp, err := p.client.CreatePayout(ctx, req)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return payoutResponseToPayment(resp)
}

func (p *Plugin) pollPayoutStatus(ctx context.Context, payoutID string) (models.PSPPayment, error) {
	resp, err := p.client.GetPayoutStatus(ctx, payoutID)
	if err != nil {
		return models.PSPPayment{}, err
	}

	// PSP returns asset in UMN format - use directly
	return payoutResponseToPayment(resp)
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

func payoutResponseToPayment(resp *client.PayoutResponse) (models.PSPPayment, error) {
	// All amounts are integers (minor units) - no decimal conversion
	var amount big.Int
	_, ok := amount.SetString(resp.Amount, 10)
	if !ok {
		return models.PSPPayment{}, fmt.Errorf("failed to parse amount %s as integer", resp.Amount)
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
		Amount:                      &amount,
		Asset:                       resp.Currency, // UMN format from PSP via currency field: "USD/2", "BTC/8"
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
// Supports all Formance payment statuses for maximum flexibility.
func mapStringToPaymentStatus(status string) models.PaymentStatus {
	switch status {
	case "PENDING":
		return models.PAYMENT_STATUS_PENDING
	case "PROCESSING":
		return models.PAYMENT_STATUS_PROCESSING
	case "SUCCEEDED":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "FAILED":
		return models.PAYMENT_STATUS_FAILED
	case "CANCELLED":
		return models.PAYMENT_STATUS_CANCELLED
	case "EXPIRED":
		return models.PAYMENT_STATUS_EXPIRED
	case "REFUNDED":
		return models.PAYMENT_STATUS_REFUNDED
	case "REFUNDED_FAILURE":
		return models.PAYMENT_STATUS_REFUNDED_FAILURE
	case "REFUND_REVERSED":
		return models.PAYMENT_STATUS_REFUND_REVERSED
	case "DISPUTE":
		return models.PAYMENT_STATUS_DISPUTE
	case "DISPUTE_WON":
		return models.PAYMENT_STATUS_DISPUTE_WON
	case "DISPUTE_LOST":
		return models.PAYMENT_STATUS_DISPUTE_LOST
	case "AUTHORISATION":
		return models.PAYMENT_STATUS_AUTHORISATION
	case "CAPTURE":
		return models.PAYMENT_STATUS_CAPTURE
	case "CAPTURE_FAILED":
		return models.PAYMENT_STATUS_CAPTURE_FAILED
	default:
		return models.PAYMENT_STATUS_OTHER
	}
}