package routable

import (
	"context"
	"errors"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/ee/plugins/routable/mappers"
	"github.com/formancehq/payments/internal/models"
)

// createPayout maps a PSPPaymentInitiation onto a Routable payable. The
// engine keeps polling via PollPayoutStatus when we return PollingPayoutID,
// which is how we honour Routable's async 202 contract without blocking.
func (p *Plugin) createPayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	pi := req.PaymentInitiation
	if err := validatePaymentInitiation(pi); err != nil {
		return models.CreatePayoutResponse{}, err
	}

	payable, err := p.initiatePayable(ctx, pi)
	if err != nil {
		return models.CreatePayoutResponse{}, err
	}

	payment, err := mappers.PayableToPSPPayment(*payable)
	if err != nil {
		return models.CreatePayoutResponse{}, fmt.Errorf("mapping payable response: %w", err)
	}

	if mappers.IsTerminalStatus(payment.Status) {
		return models.CreatePayoutResponse{Payment: &payment}, nil
	}
	id := payable.ID
	return models.CreatePayoutResponse{PollingPayoutID: &id}, nil
}

// pollPayableStatus is shared by PollPayoutStatus and PollTransferStatus.
// Both surfaces map onto the same Routable payable, so we keep a single
// implementation and dispatch from the plugin shim.
func (p *Plugin) pollPayableStatus(ctx context.Context, payableID string) (models.PollPayoutStatusResponse, error) {
	if payableID == "" {
		return models.PollPayoutStatusResponse{}, errors.New("missing payable id")
	}

	pa, err := p.client.GetPayable(ctx, payableID)
	if err != nil {
		// Eventual consistency: Routable can return 404 right after a
		// 202. Telling the engine "not yet" causes it to retry on
		// schedule.
		if errors.Is(err, client.ErrPayableNotFound) {
			return models.PollPayoutStatusResponse{}, nil
		}
		return models.PollPayoutStatusResponse{}, fmt.Errorf("polling payable %s: %w", payableID, err)
	}

	payment, err := mappers.PayableToPSPPayment(*pa)
	if err != nil {
		return models.PollPayoutStatusResponse{}, fmt.Errorf("mapping payable: %w", err)
	}

	if !mappers.IsTerminalStatus(payment.Status) {
		return models.PollPayoutStatusResponse{}, nil
	}

	if payment.Status == models.PAYMENT_STATUS_FAILED ||
		payment.Status == models.PAYMENT_STATUS_CANCELLED ||
		payment.Status == models.PAYMENT_STATUS_EXPIRED {
		errMsg := fmt.Sprintf("routable payable %s ended in %s", payableID, payment.Status)
		return models.PollPayoutStatusResponse{Error: &errMsg}, nil
	}

	return models.PollPayoutStatusResponse{Payment: &payment}, nil
}
