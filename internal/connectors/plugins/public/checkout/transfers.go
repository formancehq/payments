package checkout

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/plugins/public/checkout/client"
	"github.com/formancehq/payments/internal/models"
	
	"github.com/formancehq/go-libs/v3/currency"
)

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferPayoutRequests(pi); err != nil {
		return nil, err
	}

	tr := &client.TransferRequest{
		Reference: pi.Reference,
		Reason:    "Formance transfer",
	}
	tr.Source.EntityID = pi.SourceAccount.Reference
	tr.Source.Currency = pi.Asset
	tr.Destination.EntityID = pi.DestinationAccount.Reference
	tr.Destination.Currency = pi.Asset
	tr.Amount = pi.Amount.Int64()
	tr.IdempotencyKey = p.generateIdempotencyKey(pi.Reference)

	resp, err := p.client.InitiateTransfer(ctx, tr)
	if err != nil {
		return nil, err
	}

	raw, _ := json.Marshal(resp)

	createdAt := time.Now().UTC()
	if resp.CreatedOn != nil {
		createdAt = *resp.CreatedOn
	}

	asset := currency.FormatAsset(supportedCurrenciesWithDecimal, tr.Source.Currency)

	return &models.PSPPayment{
		ParentReference: "",
		Reference:       resp.ID,
		CreatedAt:       createdAt,
		Type:            models.PAYMENT_TYPE_TRANSFER,
		Status:          mapTransferStatusToPaymentStatus(resp.Status),
		Amount:          big.NewInt(tr.Amount),
		Asset:           asset,
		Raw:      		 raw,
	}, nil
}

func mapTransferStatusToPaymentStatus(s string) models.PaymentStatus {
	ls := strings.ToLower(strings.TrimSpace(s))
	switch ls {
	case "pending", "requested", "processing":
		return models.PAYMENT_STATUS_PENDING
	case "approved", "succeeded", "completed", "successful":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "failed", "declined", "rejected", "error":
		return models.PAYMENT_STATUS_FAILED
	case "canceled", "cancelled", "voided":
		return models.PAYMENT_STATUS_CANCELLED
	case "reversed":
		return models.PAYMENT_STATUS_REFUND_REVERSED
	default:
		return models.PAYMENT_STATUS_OTHER
	}
}

func strPtr(s string) *string { return &s }
