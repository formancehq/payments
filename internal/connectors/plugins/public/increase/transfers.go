package increase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferRequests(pi); err != nil {
		return nil, err
	}

	resp, err := p.client.InitiateTransfer(
		ctx,
		&client.TransferRequest{
			AccountID:            pi.SourceAccount.Reference,
			DestinationAccountID: pi.DestinationAccount.Reference,
			Amount:               json.Number(pi.Amount.String()),
			Description:          pi.Description,
		},
		fmt.Sprintf("transfer%s%s", pi.SourceAccount.Reference, pi.DestinationAccount.Reference),
	)
	if err != nil {
		return nil, err
	}

	return p.transferToPayment(resp)
}

func (p *Plugin) transferToPayment(transfer *client.TransferResponse) (*models.PSPPayment, error) {
	raw, err := json.Marshal(transfer)
	if err != nil {
		return nil, err
	}

	status := matchPaymentStatus(transfer.Status)

	createdAt, err := time.Parse(time.RFC3339, transfer.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse posted date %s: %w", transfer.CreatedAt, err)
	}

	precision, ok := supportedCurrenciesWithDecimal[transfer.Currency]
	if !ok {
		return nil, fmt.Errorf("unsupported currency: %s", transfer.Currency)
	}

	amount, err := currency.GetAmountWithPrecisionFromString(transfer.Amount.String(), precision)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount %s: %w", transfer.Amount, err)
	}

	return &models.PSPPayment{
		Reference:                   transfer.ID,
		CreatedAt:                   createdAt,
		Type:                        models.PAYMENT_TYPE_TRANSFER,
		Amount:                      amount,
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, transfer.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      status,
		SourceAccountReference:      &transfer.AccountID,
		DestinationAccountReference: &transfer.DestinationAccountID,
		Raw:                         raw,
	}, nil
}

func matchPaymentStatus(status string) models.PaymentStatus {
	switch status {
	case "submitted", "pending_submission", "pending_approval":
		return models.PAYMENT_STATUS_PENDING
	case "complete":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "canceled":
		return models.PAYMENT_STATUS_CANCELLED
	default:
		return models.PAYMENT_STATUS_UNKNOWN
	}
}
