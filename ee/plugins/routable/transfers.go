package routable

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/routable/mappers"
	"github.com/formancehq/payments/internal/models"
)

// createTransfer is a thin wrapper around initiatePayable. Routable does
// not distinguish "transfer" from "payout" at the API level — every
// money-out operation is a payable — so we share the create + poll
// plumbing with createPayout and only diverge at the engine response
// envelope.
func (p *Plugin) createTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	pi := req.PaymentInitiation
	if err := validatePaymentInitiation(pi); err != nil {
		return models.CreateTransferResponse{}, err
	}

	payable, err := p.initiatePayable(ctx, pi)
	if err != nil {
		return models.CreateTransferResponse{}, err
	}

	payment, err := mappers.PayableToPSPPayment(*payable)
	if err != nil {
		return models.CreateTransferResponse{}, fmt.Errorf("mapping payable response: %w", err)
	}
	// Routable's transfer surface goes through the same payable rail as
	// payouts; surface the entity as PAYMENT_TYPE_TRANSFER for the engine
	// adjustment so reporting can distinguish the two flows.
	payment.Type = models.PAYMENT_TYPE_TRANSFER

	if mappers.IsTerminalStatus(payment.Status) {
		return models.CreateTransferResponse{Payment: &payment}, nil
	}
	id := payable.ID
	return models.CreateTransferResponse{PollingTransferID: &id}, nil
}
