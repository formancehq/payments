package checkout

import (
	"context"
	"encoding/json"
	"fmt"
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

	curr, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return nil, fmt.Errorf("failed to get currency from asset %q: %w", pi.Asset, models.ErrInvalidRequest)
	}

	if !pi.Amount.IsInt64() {
		return nil, fmt.Errorf("amount overflows int64: %w", models.ErrInvalidRequest)
	}
	amountMinor := pi.Amount.Int64()

	var srcEnt, dstEnt string
	if pi.SourceAccount != nil {
		srcEnt = pi.SourceAccount.Reference
	}
	if pi.DestinationAccount != nil {
		dstEnt = pi.DestinationAccount.Reference
	}

	tr := &client.TransferRequest{
		Reference: pi.Reference,
		Reason:    "Formance transfer",
	}
	tr.Source.EntityID = srcEnt
	tr.Source.Currency = curr
	tr.Destination.EntityID = dstEnt
	tr.Destination.Currency = curr
	tr.Amount = amountMinor

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
