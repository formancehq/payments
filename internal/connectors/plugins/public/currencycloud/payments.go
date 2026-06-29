package currencycloud

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/currencycloud/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

type paymentsState struct {
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
	// LastProcessedID is the transaction ID of the last transaction emitted at
	// exactly LastUpdatedAt, so the inclusive (>=) watermark filter excludes only
	// that already-processed row while keeping distinct same-timestamp rows.
	LastProcessedID string `json:"lastProcessedID"`
	// Page is the next page to fetch within the current LastUpdatedAt second
	// (1-indexed). It advances while the watermark second is unchanged (a
	// same-second group larger than one page) and resets to 1 once the watermark
	// moves to a newer second, so a same-second group spanning pages is walked
	// without re-scanning from page 1 each cycle (which a single LastProcessedID
	// cannot do).
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
		LastUpdatedAt:   oldState.LastUpdatedAt,
		LastProcessedID: oldState.LastProcessedID,
		Page:            oldState.Page,
	}

	var payments []models.PSPPayment
	var updatedAts []time.Time
	var processedIDs []string
	hasMore := false
	// Resume at the persisted page and walk forward. We filter against the STABLE
	// oldState watermark for the whole drain (advancing it mid-pagination would
	// drop rows sharing a second across a page boundary and mutate the server
	// filter), and we do NOT trim back to PageSize — that would skip the trimmed
	// rows when resuming at the advanced page. The page cursor below records how
	// far we got.
	page := oldState.Page
	for {
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
		// Ignore shouldFetchMore's trimmed slice; keep the full over-fetch so the
		// page cursor can resume cleanly.
		needMore, hasMore, _ = shouldFetchMore(payments, nextPage, req.PageSize)

		if !needMore {
			break
		}

		page++
	}

	// Watermark and dedup key come from the last EMITTED payment, computed once
	// after the drain (updatedAts/processedIDs stay aligned with payments by index).
	if len(payments) > 0 {
		newState.LastUpdatedAt = updatedAts[len(payments)-1]
		newState.LastProcessedID = processedIDs[len(payments)-1]
		// Same-second group still draining -> resume after consumed pages; else
		// the watermark moved to a newer second, so re-anchor at page 1.
		if newState.LastUpdatedAt.Equal(oldState.LastUpdatedAt) {
			newState.Page = page + 1
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
	payments []models.PSPPayment,
	updatedAts []time.Time,
	processedIDs []string,
	pagedTransactions []client.Transaction,
	oldState paymentsState,
) ([]models.PSPPayment, []time.Time, []string, error) {
	for _, transaction := range pagedTransactions {
		// Inclusive watermark: skip transactions strictly before it, and the single
		// already-processed transaction at exactly the watermark. Distinct
		// transactions sharing that timestamp are kept (M-CON2). Keyed on
		// transaction.ID, which differs from payment.Reference (RelatedEntityID).
		cmp := transaction.UpdatedAt.Compare(oldState.LastUpdatedAt)
		if cmp < 0 || (cmp == 0 && transaction.ID == oldState.LastProcessedID) {
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
