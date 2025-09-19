package checkout

import (
	"context"
	"errors"

	"github.com/formancehq/payments/internal/connectors/plugins/public/checkout/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferPayoutRequests(pi); err != nil {
		return nil, err
	}

	var pr client.PayoutRequest
	pr.Amount = pi.Amount.Int64()
	pr.Currency = pi.Asset
	pr.Reference = pi.Reference
	pr.SourceEntityID = pi.SourceAccount.Reference
	pr.DestinationInstrumentID = pi.DestinationAccount.Reference
	pr.BillingDescriptor = pi.Description
	pr.IdempotencyKey = p.generateIdempotencyKey(pi.Reference)

	resp, err := p.client.InitiatePayout(ctx, &pr)
	if err != nil {
		return nil, err
	}

	return payoutToPayment(resp)
}

func payoutToPayment(from *client.PayoutResponse) (*models.PSPPayment, error) {
	if from == nil {
		return nil, errors.New("nil payout response")
	}

	p := &models.PSPPayment{
		Status:            mapStatus(from),
		Reference:         from.Reference,
	}
	return p, nil
}

func mapStatus(from *client.PayoutResponse) models.PaymentStatus {
	switch from.Status {
	case "Pending":
		return models.PAYMENT_STATUS_PENDING
	case "Captured":
		return models.PAYMENT_STATUS_CAPTURE
	case "Authorized", "Active":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "Declined", "Failed", "Voided":
		return models.PAYMENT_STATUS_FAILED
	case "Canceled":
		return models.PAYMENT_STATUS_CANCELLED
	default:
		return models.PAYMENT_STATUS_UNKNOWN
	}
}