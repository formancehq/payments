package moneycorp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moneycorp/client"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferPayoutRequests(pi); err != nil {
		return nil, err
	}

	curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %w", err),
			models.ErrInvalidRequest,
		)
	}

	amount, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get string amount from big int amount %v: %w", pi.Amount, err),
			models.ErrInvalidRequest,
		)
	}

	resp, err := p.client.InitiateTransfer(
		ctx,
		&client.TransferRequest{
			IdempotencyKey:     pi.Reference,
			SourceAccountID:    pi.SourceAccount.Reference,
			ReceivingAccountID: pi.DestinationAccount.Reference,
			TransferAmount:     json.Number(amount),
			TransferCurrency:   curr,
			TransferReference:  pi.Description,
			ClientReference:    pi.Description,
		},
	)
	if err != nil {
		return nil, err
	}

	return transferToPayment(resp)
}

func transferToPayment(transfer *client.TransferResponse) (*models.PSPPayment, error) {
	raw, err := json.Marshal(transfer)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction: %w", err)
	}

	createdAt, err := time.Parse("2006-01-02T15:04:05.999999999", transfer.Attributes.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction date: %w", err)
	}

	c, err := currency.GetPrecision(supportedCurrenciesWithDecimal, transfer.Attributes.TransferCurrency)
	if err != nil {
		return nil, err
	}

	amount, err := currency.GetAmountWithPrecisionFromString(transfer.Attributes.TransferAmount.String(), c)
	if err != nil {
		return nil, err
	}

	return &models.PSPPayment{
		Reference:                   transfer.ID,
		CreatedAt:                   createdAt,
		Type:                        models.PAYMENT_TYPE_TRANSFER,
		Amount:                      amount,
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, transfer.Attributes.TransferCurrency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      matchTransferStatus(transfer.Attributes.TransferStatus),
		SourceAccountReference:      pointer.For(fmt.Sprintf("%d", transfer.Attributes.SendingAccountID)),
		DestinationAccountReference: pointer.For(fmt.Sprintf("%d", transfer.Attributes.ReceivingAccountID)),
		Raw:                         raw,
	}, nil
}

func matchTransferStatus(status string) models.PaymentStatus {
	// Awaiting Dispatch, Cleared, or Cancelled
	switch status {
	case "Awaiting Dispatch":
		return models.PAYMENT_STATUS_PENDING
	case "Cleared":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "Cancelled":
		return models.PAYMENT_STATUS_FAILED
	}

	return models.PAYMENT_STATUS_UNKNOWN
}
