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

// createPayout maps a PSPPaymentInitiation onto a Routable payable.
// Routable's POST /v1/payables answers in two distinct shapes:
//
//   - 201 Created — full payable model echoed back (sync path).
//   - 202 Accepted — only {id, status: pending} (async path).
//
// We branch on the HTTP status rather than guessing from a half-mapped
// Payment. The 202 path returns PollingPayoutID immediately so the
// engine schedules PollPayoutStatus; mapping the half-empty payable
// would just throw a misleading "unsupported currency" error.
func (p *Plugin) createPayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	payable, status, err := p.initiatePayable(ctx, req.PaymentInitiation)
	if err != nil {
		return models.CreatePayoutResponse{}, err
	}

	if status == http.StatusAccepted {
		id := payable.ID
		return models.CreatePayoutResponse{PollingPayoutID: &id}, nil
	}

	// 201 Created (or any other 2xx): map the full payable.
	payment, err := mappers.PayableToPSPPayment(*payable)
	if err != nil {
		return models.CreatePayoutResponse{}, fmt.Errorf("mapping payable response: %w", err)
	}
	if mappers.IsTerminalStatus(payment.Status) {
		return models.CreatePayoutResponse{Payment: &payment}, nil
	}
	// Sync response, but Routable started us on a non-terminal status —
	// hand the engine a polling token so it converges to the terminal
	// state through PollPayoutStatus.
	id := payment.Reference
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
