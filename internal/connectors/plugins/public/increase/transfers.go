package increase

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
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

	pspPayment := &models.PSPPayment{
		ParentReference:             "",
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
	}
	pspPayment = fillReference(transfer, pspPayment)
	return pspPayment, nil
}

func fillReference(transfer *client.TransferResponse, pspPayment *models.PSPPayment) *models.PSPPayment {
	if transfer.TransactionID != "" {
		pspPayment.Reference = transfer.TransactionID
	} else if transfer.PendingTransactionID != "" {
		pspPayment.Reference = transfer.PendingTransactionID
	} else {
		pspPayment.Reference = transfer.ID
	}

	return pspPayment
}

func matchPaymentStatus(status string) models.PaymentStatus {
	status = strings.ToLower(status)
	switch status {
	case "requires_attention", "pending_reviewing",
		"pending_transfer_session_confirmation", "pending_submission",
		"pending_creating", "pending_approval", "pending_mailing":
		return models.PAYMENT_STATUS_PENDING
	case "complete", "mailed", "deposited", "submitted":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "canceled", "rejected", "stopped":
		return models.PAYMENT_STATUS_CANCELLED
	case "reversed":
		return models.PAYMENT_STATUS_REVERSED
	case "returned":
		return models.PAYMENT_STATUS_RETURNED
	default:
		return models.PAYMENT_STATUS_UNKNOWN
	}
}
