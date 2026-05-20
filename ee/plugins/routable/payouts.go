package routable

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/ee/plugins/routable/mappers"
	"github.com/formancehq/payments/internal/models"
)

// createPayout branches on Routable's HTTP status (201 sync vs 202
// async) rather than guessing from a half-mapped Payment. See
// MAPPINGS.md §5.4 for the full response-handling matrix.
func (p *Plugin) createPayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	payable, status, err := p.initiatePayable(ctx, req.PaymentInitiation)
	if err != nil {
		return models.CreatePayoutResponse{}, err
	}

	if status == http.StatusAccepted {
		id := payable.ID
		return models.CreatePayoutResponse{PollingPayoutID: &id}, nil
	}

	payment, err := mappers.PayableToPSPPayment(*payable)
	if err != nil {
		return models.CreatePayoutResponse{}, fmt.Errorf("mapping payable response: %w", err)
	}
	if mappers.IsTerminalStatus(payment.Status) {
		return models.CreatePayoutResponse{Payment: &payment}, nil
	}
	id := payment.Reference
	return models.CreatePayoutResponse{PollingPayoutID: &id}, nil
}

// pollPayableStatus is shared by PollPayoutStatus and PollTransferStatus.
func (p *Plugin) pollPayableStatus(ctx context.Context, payableID string) (models.PollPayoutStatusResponse, error) {
	if payableID == "" {
		return models.PollPayoutStatusResponse{}, errors.New("missing payable id")
	}

	pa, err := p.client.GetPayable(ctx, payableID)
	if err != nil {
		// 404 right after a 202 = Routable's eventual-consistency
		// window; tell the engine "not yet" and retry on schedule.
		if errors.Is(err, client.ErrPayableNotFound) {
			return models.PollPayoutStatusResponse{}, nil
		}
		return models.PollPayoutStatusResponse{}, fmt.Errorf("polling payable %s: %w", payableID, err)
	}

	payment, err := mappers.PayableToPSPPayment(*pa)
	if err != nil {
		return models.PollPayoutStatusResponse{}, fmt.Errorf("mapping payable: %w", err)
	}
	// Link PI ↔ Payment now; FETCH_PAYMENTS picks up further transitions.
	return models.PollPayoutStatusResponse{Payment: &payment}, nil
}
