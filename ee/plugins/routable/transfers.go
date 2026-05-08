package routable

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/ee/plugins/routable/mappers"
	"github.com/formancehq/payments/internal/models"
)

// createTransfer is a thin wrapper around initiatePayable. Routable does
// not distinguish "transfer" from "payout" at the API level — every
// money-out operation is a payable — so we share the create + poll
// plumbing with createPayout. The only thing that differs is which
// engine response envelope wraps the result. Same 201/202 branching as
// createPayout (see payouts.go for the rationale).
func (p *Plugin) createTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	payable, status, err := p.initiatePayable(ctx, req.PaymentInitiation)
	if err != nil {
		return models.CreateTransferResponse{}, err
	}

	if status == http.StatusAccepted {
		id := payable.ID
		return models.CreateTransferResponse{PollingTransferID: &id}, nil
	}

	payment, err := mappers.PayableToPSPPayment(*payable)
	if err != nil {
		return models.CreateTransferResponse{}, fmt.Errorf("mapping payable response: %w", err)
	}
	// Routable's transfer surface goes through the same payable rail as
	// payouts; surface the entity as PAYMENT_TYPE_TRANSFER for the
	// engine adjustment so reporting can distinguish the two flows.
	payment.Type = models.PAYMENT_TYPE_TRANSFER

	if mappers.IsTerminalStatus(payment.Status) {
		return models.CreateTransferResponse{Payment: &payment}, nil
	}
	id := payment.Reference
	return models.CreateTransferResponse{PollingTransferID: &id}, nil
}
