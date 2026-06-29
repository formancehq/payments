package generic

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/payments/genericclient/v3"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
)

type paymentsState struct {
	LastUpdatedAtFrom time.Time `json:"lastUpdatedAtFrom"`
	// LastProcessedID is the reference of the last payment emitted at exactly
	// LastUpdatedAtFrom, so the inclusive (>=) watermark filter can exclude only
	// that already-processed row while keeping distinct same-timestamp payments.
	LastProcessedID string `json:"lastProcessedID"`
	// Page is the next page to fetch within the current LastUpdatedAtFrom second.
	// It advances while the watermark second is unchanged (a same-second group
	// larger than one page) and resets to 1 once the watermark moves to a newer
	// second, so a same-second group spanning pages is walked without re-scanning
	// from page 1 each cycle (which a single LastProcessedID cannot do).
	Page int `json:"page"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}
	if oldState.Page < 1 {
		oldState.Page = 1
	}

	newState := paymentsState{
		LastUpdatedAtFrom: oldState.LastUpdatedAtFrom,
		LastProcessedID:   oldState.LastProcessedID,
		Page:              oldState.Page,
	}

	payments := make([]models.PSPPayment, 0)
	updatedAts := make([]time.Time, 0)
	needMore := false
	hasMore := false
	// Resume at the persisted page and walk forward. We do NOT trim back to
	// PageSize and restart at page 1 next cycle: that would skip the trimmed
	// rows. Instead the page cursor (below) records how far we got.
	page := oldState.Page
	for {
		pagedPayments, err := p.client.ListTransactions(ctx, int64(page), int64(req.PageSize), oldState.LastUpdatedAtFrom)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		payments, updatedAts, err = fillPayments(pagedPayments, payments, updatedAts, oldState)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(payments, pagedPayments, req.PageSize)
		if !needMore || !hasMore {
			break
		}
		page++
	}

	if len(updatedAts) > 0 {
		newState.LastUpdatedAtFrom = updatedAts[len(updatedAts)-1]
		newState.LastProcessedID = payments[len(payments)-1].Reference
		// While the watermark second is unchanged the batch was one same-second
		// group: advance past the consumed pages only if there is definitely a
		// full next page (hasMore). If it drained on a short final page, keep the
		// cursor there — a newer row appended to that second's >= watermark query
		// lands on this very page, so advancing past it would strand it forever.
		// When the watermark moved to a newer second, re-anchor at page 1.
		if newState.LastUpdatedAtFrom.Equal(oldState.LastUpdatedAtFrom) {
			if hasMore {
				newState.Page = page + 1
			} else {
				newState.Page = page
			}
		} else {
			newState.Page = 1
		}
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func fillPayments(
	pagedPayments []genericclient.Transaction,
	payments []models.PSPPayment,
	updatedAts []time.Time,
	oldState paymentsState,
) ([]models.PSPPayment, []time.Time, error) {
	for _, payment := range pagedPayments {
		// Inclusive watermark: skip payments strictly before it, and the single
		// already-processed payment at exactly the watermark. Distinct payments
		// sharing that timestamp are kept (M-CON2).
		cmp := payment.UpdatedAt.Compare(oldState.LastUpdatedAtFrom)
		if cmp < 0 || (cmp == 0 && payment.Id == oldState.LastProcessedID) {
			continue
		}

		raw, err := json.Marshal(payment)
		if err != nil {
			return nil, nil, err
		}

		paymentType := matchPaymentType(payment.Type)
		paymentStatus := matchPaymentStatus(payment.Status)

		var amount big.Int
		_, ok := amount.SetString(payment.Amount, 10)
		if !ok {
			return nil, nil, fmt.Errorf("failed to parse amount: %s", payment.Amount)
		}

		p := models.PSPPayment{
			Reference: payment.Id,
			CreatedAt: payment.CreatedAt,
			Type:      paymentType,
			Amount:    &amount,
			Asset:     payment.Currency, // UMN format from PSP: "USD/2", "BTC/8"
			Scheme:    models.PAYMENT_SCHEME_OTHER,
			Status:    paymentStatus,
			Metadata:  payment.Metadata,
			Raw:       raw,
		}

		if payment.RelatedTransactionID != nil {
			p.ParentReference = *payment.RelatedTransactionID
		}

		if payment.SourceAccountID != nil {
			p.SourceAccountReference = payment.SourceAccountID
		}

		if payment.DestinationAccountID != nil {
			p.DestinationAccountReference = payment.DestinationAccountID
		}

		payments = append(payments, p)
		updatedAts = append(updatedAts, payment.UpdatedAt)
	}

	return payments, updatedAts, nil
}

func matchPaymentType(
	transactionType genericclient.TransactionType,
) models.PaymentType {
	switch transactionType {
	case genericclient.PAYIN:
		return models.PAYMENT_TYPE_PAYIN
	case genericclient.PAYOUT:
		return models.PAYMENT_TYPE_PAYOUT
	case genericclient.TRANSFER:
		return models.PAYMENT_TYPE_TRANSFER
	default:
		return models.PAYMENT_TYPE_OTHER
	}
}

func matchPaymentStatus(
	status genericclient.TransactionStatus,
) models.PaymentStatus {
	switch status {
	case genericclient.PENDING:
		return models.PAYMENT_STATUS_PENDING
	case genericclient.SUCCEEDED:
		return models.PAYMENT_STATUS_SUCCEEDED
	case genericclient.FAILED:
		return models.PAYMENT_STATUS_FAILED
	case genericclient.CANCELLED:
		return models.PAYMENT_STATUS_CANCELLED
	case genericclient.EXPIRED:
		return models.PAYMENT_STATUS_EXPIRED
	case genericclient.REFUNDED:
		return models.PAYMENT_STATUS_REFUNDED
	case genericclient.REFUNDED_FAILURE:
		return models.PAYMENT_STATUS_REFUNDED_FAILURE
	case genericclient.REFUND_REVERSED:
		return models.PAYMENT_STATUS_REFUND_REVERSED
	case genericclient.DISPUTE:
		return models.PAYMENT_STATUS_DISPUTE
	case genericclient.DISPUTE_WON:
		return models.PAYMENT_STATUS_DISPUTE_WON
	case genericclient.DISPUTE_LOST:
		return models.PAYMENT_STATUS_DISPUTE_LOST
	case genericclient.AUTHORISATION:
		return models.PAYMENT_STATUS_AUTHORISATION
	case genericclient.CAPTURE:
		return models.PAYMENT_STATUS_CAPTURE
	case genericclient.CAPTURE_FAILED:
		return models.PAYMENT_STATUS_CAPTURE_FAILED
	case genericclient.OTHER:
		return models.PAYMENT_STATUS_OTHER
	default:
		return models.PAYMENT_STATUS_OTHER
	}
}
