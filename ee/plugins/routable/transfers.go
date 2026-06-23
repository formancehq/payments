package routable

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/ee/plugins/routable/mappers"
	"github.com/formancehq/payments/pkg/domain/models"
)

// createTransfer mirrors createPayout: Routable maps both flows onto the
// same payable rail. We override Type to PAYMENT_TYPE_TRANSFER so the
// engine adjustment can distinguish payouts from transfers in reporting.
// 201/202 branching is identical to createPayout (see payouts.go).
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
	payment.Type = models.PAYMENT_TYPE_TRANSFER

	if mappers.IsTerminalStatus(payment.Status) {
		return models.CreateTransferResponse{Payment: &payment}, nil
	}
	id := payment.Reference
	return models.CreateTransferResponse{PollingTransferID: &id}, nil
}
