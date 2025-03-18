package increase

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferRequests(pi); err != nil {
		return nil, err
	}

	idempotencyKey := p.generateIdempotencyKey(pi.Reference)
	resp, err := p.client.InitiateTransfer(
		ctx,
		&client.TransferRequest{
			AccountID:            pi.SourceAccount.Reference,
			DestinationAccountID: pi.DestinationAccount.Reference,
			Amount:               pi.Amount.Int64(),
			Description:          pi.Description,
		},
		idempotencyKey,
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

	return &models.PSPPayment{
		Reference:                   transfer.ID,
		CreatedAt:                   createdAt,
		Type:                        models.PAYMENT_TYPE_TRANSFER,
		Amount:                      big.NewInt(transfer.Amount),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, transfer.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      status,
		SourceAccountReference:      &transfer.AccountID,
		DestinationAccountReference: &transfer.DestinationAccountID,
		Raw:                         raw,
		Metadata: map[string]string{
			client.IncreaseDescriptionMetadataKey:              transfer.Description,
			client.IncreaseTransactionIDMetadataKey:            transfer.TransactionID,
			client.IncreaseDestinationTransactionIDMetadataKey: transfer.DestinationTransactionID,
		},
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
