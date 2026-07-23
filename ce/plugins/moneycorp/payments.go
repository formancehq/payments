package moneycorp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	"github.com/formancehq/payments/ce/plugins/moneycorp/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
	"github.com/formancehq/payments/pkg/domain/plugins"
)

type paymentsState struct {
	LastCreatedAt time.Time `json:"lastCreatedAt"`
	// LastProcessedIDs holds the transaction IDs of ALL transactions already
	// emitted at exactly LastCreatedAt, accumulated across cycles while the
	// watermark second is unchanged and reset when it advances. The server filter
	// is inclusive (>=), so each cycle rescans from page 0 and skips this whole
	// set: a same-second group larger than PageSize is walked across cycles
	// without a drifting page cursor, and a multi-row final page cannot oscillate
	// (every already-emitted sibling is skipped, not just one). Keyed on
	// transaction.ID (the iteration identity), which differs from
	// payment.Reference for "Payment" types.
	//
	// Migration note: the old singular lastProcessedID and page fields are
	// ignored. After deploy the watermark second is re-emitted once (idempotent —
	// storage upserts dedup it) and no recrawl occurs.
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

	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextPaymentsResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	newState := paymentsState{
		LastCreatedAt:    oldState.LastCreatedAt,
		LastProcessedIDs: oldState.LastProcessedIDs,
	}

	payments := make([]models.PSPPayment, 0, req.PageSize)
	processedIDs := make([]string, 0, req.PageSize)
	needMore := false
	hasMore := false
	// Rescan from page 0 each cycle (no page cursor): the processed-ID set skips
	// every already-emitted sibling at the watermark second, so a same-second
	// group larger than PageSize is walked across cycles and a multi-row final
	// page cannot oscillate. The server filter is inclusive (>=), so page 0
	// re-includes the watermark second.
	for page := 0; ; page++ {
		pagedTransactions, err := p.client.GetTransactions(ctx, from.Reference, page, req.PageSize, oldState.LastCreatedAt)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		payments, processedIDs, err = p.toPSPPayments(ctx, oldState.LastCreatedAt, oldState.LastProcessedIDs, payments, processedIDs, pagedTransactions)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(payments, pagedTransactions, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	if len(payments) > 0 {
		last := payments[len(payments)-1].CreatedAt

		// Collect the transaction IDs emitted at exactly the new watermark second.
		idsAtWatermark := make([]string, 0)
		for i := range payments {
			if payments[i].CreatedAt.Equal(last) {
				idsAtWatermark = append(idsAtWatermark, processedIDs[i])
			}
		}

		// Accumulate the processed-ID set while still inside the same watermark
		// second; reset it when the watermark advances to a newer second.
		if last.Equal(oldState.LastCreatedAt) {
			newState.LastProcessedIDs = append(oldState.LastProcessedIDs, idsAtWatermark...)
		} else {
			newState.LastProcessedIDs = idsAtWatermark
		}
		newState.LastCreatedAt = last
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

func (p *Plugin) toPSPPayments(
	ctx context.Context,
	lastCreatedAt time.Time,
	lastProcessedIDs []string,
	payments []models.PSPPayment,
	processedIDs []string,
	transactions []*client.Transaction,
) ([]models.PSPPayment, []string, error) {
	for _, transaction := range transactions {
		createdAt, err := time.Parse("2006-01-02T15:04:05.999999999", transaction.Attributes.CreatedAt)
		if err != nil {
			return payments, processedIDs, fmt.Errorf("failed to parse transaction date: %v", err)
		}

		// Inclusive watermark: skip transactions strictly before it, and any
		// already-emitted transaction at exactly the watermark second. Distinct
		// transactions sharing that timestamp are kept (M-CON2).
		cmp := createdAt.Compare(lastCreatedAt)
		if cmp < 0 || (cmp == 0 && slices.Contains(lastProcessedIDs, transaction.ID)) {
			continue
		}

		payment, err := p.transactionToPayment(ctx, transaction)
		if err != nil {
			if errors.Is(err, plugins.ErrCurrencyNotSupported) {
				// Skip unsupported currencies rather than failing: a retryable
				// error here would freeze ingestion for the whole account.
				p.logger.WithField("transaction_id", transaction.ID).Info("skipping transaction with unsupported currency")
				continue
			}
			return payments, processedIDs, err
		}
		if payment == nil {
			continue
		}

		payments = append(payments, *payment)
		processedIDs = append(processedIDs, transaction.ID)
	}
	return payments, processedIDs, nil
}

func (p *Plugin) transactionToPayment(ctx context.Context, transaction *client.Transaction) (*models.PSPPayment, error) {
	switch transaction.Attributes.Type {
	case "Transfer":
		if transaction.Attributes.Direction == "Debit" {
			return p.fetchAndTranslateTransfer(ctx, transaction)
		} else {
			// Do not fetch the transfer, it does not exists if we're trying to
			// fetch it with the destination account.
			return nil, nil
		}
	}

	rawData, err := json.Marshal(transaction)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction: %w", err)
	}

	paymentType, shouldBeRecorded := matchPaymentType(transaction.Attributes.Type, transaction.Attributes.Direction)
	if !shouldBeRecorded {
		return nil, nil
	}

	createdAt, err := time.Parse("2006-01-02T15:04:05.999999999", transaction.Attributes.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction date: %w", err)
	}

	c, err := currency.GetPrecision(supportedCurrenciesWithDecimal, transaction.Attributes.Currency)
	if err != nil {
		if errors.Is(err, currency.ErrMissingCurrencies) {
			return nil, fmt.Errorf("%w: %w", plugins.ErrCurrencyNotSupported, err)
		}
		return nil, err
	}

	amount, err := currency.GetAmountWithPrecisionFromString(transaction.Attributes.Amount.String(), c)
	if err != nil {
		return nil, err
	}

	reference := transaction.ID
	if transaction.Attributes.Type == "Payment" {
		// In case of payments (related to payouts), we want to take the real
		// object id as a reference
		reference = transaction.Relationships.Data.ID
	}

	payment := models.PSPPayment{
		Reference: reference,
		CreatedAt: createdAt,
		Type:      paymentType,
		Amount:    amount,
		Asset:     currency.FormatAsset(supportedCurrenciesWithDecimal, transaction.Attributes.Currency),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    models.PAYMENT_STATUS_SUCCEEDED,
		Raw:       rawData,
	}

	switch paymentType {
	case models.PAYMENT_TYPE_PAYIN:
		payment.DestinationAccountReference = pointer.For(strconv.Itoa(int(transaction.Attributes.AccountID)))
	case models.PAYMENT_TYPE_PAYOUT:
		payment.SourceAccountReference = pointer.For(strconv.Itoa(int(transaction.Attributes.AccountID)))
	default:
		if transaction.Attributes.Direction == "Debit" {
			payment.SourceAccountReference = pointer.For(strconv.Itoa(int(transaction.Attributes.AccountID)))
		} else {
			payment.DestinationAccountReference = pointer.For(strconv.Itoa(int(transaction.Attributes.AccountID)))
		}
	}

	return &payment, nil
}

func (p *Plugin) fetchAndTranslateTransfer(ctx context.Context, transaction *client.Transaction) (*models.PSPPayment, error) {
	transfer, err := p.client.GetTransfer(ctx, fmt.Sprint(transaction.Attributes.AccountID), transaction.Relationships.Data.ID)
	if err != nil {
		return nil, err
	}

	return transferToPayment(transfer)
}

func matchPaymentType(transactionType string, transactionDirection string) (models.PaymentType, bool) {
	switch transactionType {
	case "Transfer":
		return models.PAYMENT_TYPE_TRANSFER, true
	case "Payment", "Exchange", "Charge", "Refund":
		switch transactionDirection {
		case "Debit":
			return models.PAYMENT_TYPE_PAYOUT, true
		case "Credit":
			return models.PAYMENT_TYPE_PAYIN, true
		}
	}

	return models.PAYMENT_TYPE_OTHER, false
}
