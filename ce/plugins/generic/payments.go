package generic

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"
	"time"

	"github.com/formancehq/payments/ce/plugins/generic/client/generated"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
)

type paymentsState struct {
	LastUpdatedAtFrom time.Time `json:"lastUpdatedAtFrom"`
	// LastProcessedIDs holds the references of ALL payments already emitted at
	// exactly LastUpdatedAtFrom, accumulated across cycles while the watermark
	// second is unchanged and reset when it advances. The server filter is
	// inclusive (>=), so each cycle rescans from page 1 and skips this whole set:
	// a same-second group larger than PageSize is walked across cycles without a
	// drifting page cursor, and a multi-row final page cannot oscillate (every
	// already-emitted sibling is skipped, not just one).
	//
	// Precision: comparison and the ID set use the exact timestamp the API
	// returns (full precision, as in the qonto reference), not a truncated
	// second; "same-second" above is shorthand because these PSPs report
	// timestamps at second granularity, so equal values represent the same second.
	LastProcessedIDs []string `json:"lastProcessedIDs"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	newState := paymentsState{
		LastUpdatedAtFrom: oldState.LastUpdatedAtFrom,
		LastProcessedIDs:  oldState.LastProcessedIDs,
	}

	payments := make([]models.PSPPayment, 0)
	updatedAts := make([]time.Time, 0)
	needMore := false
	hasMore := false
	// Rescan from page 1 each cycle (no page cursor): the processed-ID set skips
	// every already-emitted sibling at the watermark second, so a same-second
	// group larger than PageSize is walked across cycles and a multi-row final
	// page cannot oscillate. The server filter is inclusive (>=), so page 1
	// re-includes the watermark second.
	for page := 1; ; page++ {
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
	}

	if len(updatedAts) > 0 {
		lastUpdatedAt := updatedAts[len(updatedAts)-1]

		// Collect the references emitted at exactly the new watermark second.
		idsAtWatermark := make([]string, 0)
		for i := range payments {
			if updatedAts[i].Equal(lastUpdatedAt) {
				idsAtWatermark = append(idsAtWatermark, payments[i].Reference)
			}
		}

		// Accumulate the processed-ID set while still inside the same watermark
		// second; reset it when the watermark advances to a newer second.
		if lastUpdatedAt.Equal(oldState.LastUpdatedAtFrom) {
			newState.LastProcessedIDs = append(oldState.LastProcessedIDs, idsAtWatermark...)
		} else {
			newState.LastProcessedIDs = idsAtWatermark
		}
		newState.LastUpdatedAtFrom = lastUpdatedAt
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
		// Inclusive watermark: skip payments strictly before it, and any already-
		// emitted payment at exactly the watermark second. Distinct payments
		// sharing that timestamp are kept (M-CON2).
		cmp := payment.UpdatedAt.Compare(oldState.LastUpdatedAtFrom)
		if cmp < 0 || (cmp == 0 && slices.Contains(oldState.LastProcessedIDs, payment.Id)) {
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
