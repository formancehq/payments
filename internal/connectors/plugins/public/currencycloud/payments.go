package currencycloud

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/currencycloud/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

type paymentsState struct {
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
	// LastProcessedIDs holds the transaction IDs of ALL transactions already
	// emitted at exactly LastUpdatedAt, accumulated across cycles while the
	// watermark second is unchanged and reset when it advances. The server filter
	// is inclusive (>=), so each cycle rescans from page 1 and skips this whole
	// set: a same-second group larger than PageSize is walked across cycles
	// without a drifting page cursor, and a multi-row final page cannot oscillate
	// (every already-emitted sibling is skipped, not just one).
	//
	// Migration: the old singular lastProcessedID and page/lastPage fields are
	// ignored. After deploy the watermark second is re-emitted once (idempotent
	// via storage upserts), with no recrawl.
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
		LastUpdatedAt:    oldState.LastUpdatedAt,
		LastProcessedIDs: oldState.LastProcessedIDs,
	}

	var payments []models.PSPPayment
	var updatedAts []time.Time
	var processedIDs []string
	hasMore := false
	// Rescan from page 1 each cycle (no page cursor): the processed-ID set skips
	// every already-emitted sibling at the watermark second, so a same-second
	// group larger than PageSize is walked across cycles and a multi-row final
	// page cannot oscillate. We filter against the STABLE oldState watermark for
	// the whole drain, and we do NOT trim back to PageSize.
	for page := 1; ; page++ {
		pagedTransactions, nextPage, err := p.client.GetTransactions(ctx, page, req.PageSize, oldState.LastUpdatedAt)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		if len(pagedTransactions) == 0 {
			break
		}

		payments, updatedAts, processedIDs, err = fillPayments(payments, updatedAts, processedIDs, pagedTransactions, oldState)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		var needMore bool
		// Ignore shouldFetchMore's trimmed slice; keep the full over-fetch.
		needMore, hasMore, _ = shouldFetchMore(payments, nextPage, req.PageSize)

		if !needMore {
			break
		}
	}

	// Watermark and dedup set come from the last EMITTED payment, computed once
	// after the drain (updatedAts/processedIDs stay aligned with payments by index).
	if len(updatedAts) > 0 {
		last := updatedAts[len(updatedAts)-1]

		// Collect the IDs emitted at exactly the new watermark second.
		idsAtWatermark := make([]string, 0)
		for i := range processedIDs {
			if updatedAts[i].Equal(last) {
				idsAtWatermark = append(idsAtWatermark, processedIDs[i])
			}
		}

		// Accumulate the processed-ID set while still inside the same watermark
		// second; reset it when the watermark advances to a newer second.
		if last.Equal(oldState.LastUpdatedAt) {
			newState.LastProcessedIDs = append(oldState.LastProcessedIDs, idsAtWatermark...)
		} else {
			newState.LastProcessedIDs = idsAtWatermark
		}
		newState.LastUpdatedAt = last
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
	payments []models.PSPPayment,
	updatedAts []time.Time,
	processedIDs []string,
	pagedTransactions []client.Transaction,
	oldState paymentsState,
) ([]models.PSPPayment, []time.Time, []string, error) {
	for _, transaction := range pagedTransactions {
		// Inclusive watermark: skip transactions strictly before it, and any
		// already-emitted transaction at exactly the watermark second. Distinct
		// transactions sharing that timestamp are kept (M-CON2). Keyed on
		// transaction.ID, which differs from payment.Reference (RelatedEntityID).
		cmp := transaction.UpdatedAt.Compare(oldState.LastUpdatedAt)
		if cmp < 0 || (cmp == 0 && slices.Contains(oldState.LastProcessedIDs, transaction.ID)) {
			continue
		}

		payment, err := transactionToPayment(transaction)
		if err != nil {
			return nil, nil, nil, err
		}

		if payment != nil {
			payments = append(payments, *payment)
			updatedAts = append(updatedAts, transaction.UpdatedAt)
			processedIDs = append(processedIDs, transaction.ID)
		}
	}

	return payments, updatedAts, processedIDs, nil
}

func transactionToPayment(transaction client.Transaction) (*models.PSPPayment, error) {
	raw, err := json.Marshal(transaction)
	if err != nil {
		return nil, err
	}

	precision, ok := supportedCurrenciesWithDecimal[transaction.Currency]
	if !ok {
		return nil, nil
	}

	amount, err := currency.GetAmountWithPrecisionFromString(transaction.Amount.String(), precision)
	if err != nil {
		return nil, err
	}

	paymentType := matchTransactionType(transaction.RelatedEntityType, transaction.Type)

	reference := transaction.RelatedEntityID
	if reference == "" {
		reference = transaction.ID
	}
	payment := &models.PSPPayment{
		Reference: reference,
		CreatedAt: transaction.CreatedAt,
		Type:      paymentType,
		Amount:    amount,
		Asset:     currency.FormatAsset(supportedCurrenciesWithDecimal, transaction.Currency),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    matchTransactionStatus(transaction.Status),
		Raw:       raw,
	}

	switch paymentType {
	case models.PAYMENT_TYPE_PAYOUT:
		payment.SourceAccountReference = &transaction.AccountID
	case models.PAYMENT_TYPE_PAYIN:
		payment.DestinationAccountReference = &transaction.AccountID
	}

	return payment, nil
}

func matchTransactionType(entityType string, transactionType string) models.PaymentType {
	switch entityType {
	case "inbound_funds":
		return models.PAYMENT_TYPE_PAYIN
	case "payment":
		return models.PAYMENT_TYPE_PAYOUT
	case "transfer", "balance_transfer":
		return models.PAYMENT_TYPE_TRANSFER
	default:
		switch transactionType {
		case "credit":
			return models.PAYMENT_TYPE_PAYIN
		case "debit":
			return models.PAYMENT_TYPE_PAYOUT
		}
	}

	return models.PAYMENT_TYPE_OTHER
}

func matchTransactionStatus(transactionStatus string) models.PaymentStatus {
	switch transactionStatus {
	case "completed":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "pending", "ready_to_send":
		return models.PAYMENT_STATUS_PENDING
	case "deleted":
		return models.PAYMENT_STATUS_FAILED
	case "cancelled":
		return models.PAYMENT_STATUS_CANCELLED
	}
	return models.PAYMENT_STATUS_OTHER
}
