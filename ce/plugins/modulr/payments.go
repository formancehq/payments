package modulr

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/payments/ce/plugins/modulr/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

// transactionDateLayout is the timestamp layout Modulr uses for transaction/posted dates.
const transactionDateLayout = "2006-01-02T15:04:05.999-0700"

// paymentsStateVersion is the current schema version of paymentsState. Version 0 derived
// the watermark from PostedDate but filtered/compared by TransactionDate; version 1 keyed
// everything on TransactionDate; version 2 keys everything on PostedDate — the field the
// endpoint actually sorts and paginates by (verified against the sandbox: postedDate is
// strictly descending, transactionDate is not). State written under an older version is
// reset once on read so the corrected logic re-ingests history (idempotent by reference).
const paymentsStateVersion = 2

// The Modulr transactions endpoint returns results newest-first BY POSTEDDATE and exposes
// no sort parameter, while the engine must ingest payments oldest-first: it seeds a
// payment's base row from the first adjustment it sees for a reference (storage upserts the
// row with ON CONFLICT DO NOTHING). PostedDate is also the payment's CreatedAt, so keying
// the window on it aligns the sort key, the fromPostedDate/toPostedDate filter, the ceiling
// and the watermark on one immutable, strictly-ordered field. So each poll drains a frozen
// window (LastPostedTime, Ceiling] one page at a time across successive calls:
//
//  1. open    — peek the newest page (page 0) to freeze Ceiling and read TotalPages, which
//     tells us the oldest page index.
//  2. descend — emit pages from the oldest (TotalPages-1) up to page 0, reversing each page
//     so the whole window is emitted oldest-first; commit the watermark to Ceiling only
//     after page 0.
//
// Every call returns at most ~PageSize payments (Temporal-safe) and the watermark advances
// only once the window is fully drained, so an interrupted drain simply re-runs the window
// (idempotent by reference).
type paymentsState struct {
	// LastPostedTime is the committed watermark: the greatest postedDate already ingested.
	// Advanced only when a drain window is fully consumed.
	LastPostedTime time.Time `json:"lastPostedTime"`
	// Ceiling is the frozen upper bound (postedDate) of the drain window currently
	// in progress, and the sole indicator that a window is in progress. A zero Ceiling
	// means no window is in progress: the next call opens a fresh one by peeking the
	// newest row. Because the window only ever commits a non-zero Ceiling, we can never
	// page an unbounded window or commit a zero watermark.
	Ceiling time.Time `json:"ceiling"`
	// NextPage is the next page index to fetch within the in-progress window. It counts
	// DOWN from the oldest page to 0 so transactions are emitted oldest-first.
	NextPage int `json:"nextPage"`
	// Version is the state schema version (see paymentsStateVersion).
	Version int `json:"version"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var state paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	// Migration: state written under an older version keyed the watermark on a different
	// field (v0 PostedDate-derived but TransactionDate-filtered; v1 TransactionDate).
	// Reset it once so the corrected postedDate logic re-ingests the available history
	// (payments are idempotent by reference).
	if state.Version < paymentsStateVersion {
		state = paymentsState{Version: paymentsStateVersion}
	}

	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextPaymentsResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	// A non-zero Ceiling means a window is mid-drain; otherwise open a fresh one.
	if state.Ceiling.IsZero() {
		return p.openPaymentsWindow(ctx, req, from, state)
	}
	return p.drainPaymentsWindow(ctx, req, from, state)
}

// openPaymentsWindow peeks the newest page to freeze the window ceiling (the watermark we
// commit once drained) and reads TotalPages to find the oldest page. A single page is
// emitted straight away; a multi-page window starts the oldest-first descent.
func (p *Plugin) openPaymentsWindow(ctx context.Context, req models.FetchNextPaymentsRequest, from models.PSPAccount, state paymentsState) (models.FetchNextPaymentsResponse, error) {
	page0, totalPages, err := p.client.GetTransactions(ctx, from.Reference, 0, req.PageSize, state.LastPostedTime, time.Time{})
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}
	if len(page0) == 0 {
		// Nothing newer than the watermark; stay put.
		return marshalPaymentsResponse(state, nil, false)
	}

	// Results are newest-first by postedDate, so the first row carries the greatest
	// postedDate — the true window ceiling.
	state.Ceiling, err = time.Parse(transactionDateLayout, page0[0].PostedDate)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to parse posted date %s: %w", page0[0].PostedDate, err)
	}

	if totalPages <= 1 {
		// The only page is both newest and oldest: emit it and close the window.
		payments, err := p.paymentsFromPage(ctx, page0, from, state.LastPostedTime, state.Ceiling)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
		return marshalPaymentsResponse(closeWindow(state), payments, false)
	}

	// Multi-page: descend from the oldest page. Page 0 (just peeked) is the newest and is
	// emitted last, so we don't emit it here.
	state.NextPage = totalPages - 1
	return marshalPaymentsResponse(state, nil, true)
}

// drainPaymentsWindow emits one page of the frozen window, walking from the oldest page
// down to page 0, and commits the watermark to the ceiling once page 0 has been emitted.
func (p *Plugin) drainPaymentsWindow(ctx context.Context, req models.FetchNextPaymentsRequest, from models.PSPAccount, state paymentsState) (models.FetchNextPaymentsResponse, error) {
	// Bound above by the frozen ceiling so transactions arriving mid-drain don't shift
	// page indices.
	page, _, err := p.client.GetTransactions(ctx, from.Reference, state.NextPage, req.PageSize, state.LastPostedTime, state.Ceiling)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments, err := p.paymentsFromPage(ctx, page, from, state.LastPostedTime, state.Ceiling)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	if state.NextPage > 0 {
		// Older pages already emitted; keep descending towards the newest page.
		state.NextPage--
		return marshalPaymentsResponse(state, payments, true)
	}

	// Page 0 (newest) just emitted: the window is fully drained.
	return marshalPaymentsResponse(closeWindow(state), payments, false)
}

// paymentsFromPage maps one newest-first page of transactions into payments inside the
// window (lower, upper], emitted oldest-first. The window is keyed on postedDate, the
// field the endpoint sorts by.
func (p *Plugin) paymentsFromPage(ctx context.Context, page []client.Transaction, from models.PSPAccount, lower, upper time.Time) ([]models.PSPPayment, error) {
	payments := make([]models.PSPPayment, 0, len(page))
	for _, transaction := range reverseTransactions(page) {
		date, err := time.Parse(transactionDateLayout, transaction.PostedDate)
		if err != nil {
			return nil, err
		}

		// Keep only transactions inside the window (lower, upper]: skip those already
		// ingested (<= lower) or newer than the frozen ceiling (> upper).
		if !date.After(lower) || date.After(upper) {
			continue
		}

		payment, err := p.transactionToPayment(ctx, transaction, from)
		if err != nil {
			return nil, err
		}
		if payment != nil {
			payments = append(payments, *payment)
		}
	}

	return payments, nil
}

// reverseTransactions returns the page oldest-first; Modulr returns it newest-first by
// postedDate.
func reverseTransactions(in []client.Transaction) []client.Transaction {
	out := make([]client.Transaction, len(in))
	for i := range in {
		out[len(in)-1-i] = in[i]
	}
	return out
}

// closeWindow returns state with the watermark advanced to the frozen ceiling and the
// drain window cleared, so the next poll opens a fresh window.
func closeWindow(state paymentsState) paymentsState {
	state.LastPostedTime = state.Ceiling
	state.Ceiling = time.Time{}
	state.NextPage = 0
	return state
}

func marshalPaymentsResponse(state paymentsState, payments []models.PSPPayment, hasMore bool) (models.FetchNextPaymentsResponse, error) {
	payload, err := json.Marshal(state)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}
	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func (p *Plugin) transactionToPayment(
	ctx context.Context,
	transaction client.Transaction,
	from models.PSPAccount,
) (*models.PSPPayment, error) {
	raw, err := json.Marshal(transaction)
	if err != nil {
		return nil, err
	}

	paymentType := matchTransactionType(transaction.Type)
	switch paymentType {
	case models.PAYMENT_TYPE_TRANSFER:
		// We want to fetch the transfer details in order to have the source
		// and destination account references
		return p.fetchAndTranslateTransfer(ctx, transaction)
	default:
	}

	precision, ok := supportedCurrenciesWithDecimal[transaction.Account.Currency]
	if !ok {
		return nil, nil
	}

	amount, err := currency.GetAmountWithPrecisionFromString(transaction.Amount.String(), precision)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount %s: %w", transaction.Amount, err)
	}

	createdAt, err := time.Parse("2006-01-02T15:04:05.999-0700", transaction.PostedDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse posted date %s: %w", transaction.PostedDate, err)
	}

	payment := &models.PSPPayment{
		Reference: transaction.SourceID, // Do not take the transaction ID, but the source ID
		CreatedAt: createdAt,
		Type:      paymentType,
		Amount:    amount,
		Asset:     currency.FormatAsset(supportedCurrenciesWithDecimal, transaction.Account.Currency),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    models.PAYMENT_STATUS_SUCCEEDED,
		Raw:       raw,
	}

	switch paymentType {
	case models.PAYMENT_TYPE_PAYIN:
		payment.DestinationAccountReference = &from.Reference
	case models.PAYMENT_TYPE_PAYOUT:
		payment.SourceAccountReference = &from.Reference
	default:
		if transaction.Credit {
			payment.DestinationAccountReference = &from.Reference
		} else {
			payment.SourceAccountReference = &from.Reference
		}
	}

	return payment, nil
}

func (p *Plugin) fetchAndTranslateTransfer(
	ctx context.Context,
	transaction client.Transaction,
) (*models.PSPPayment, error) {
	if !transaction.Credit {
		// Transfer are reprensented as double transactions: one for the source
		// account and one for the destination account. We don't want to generate
		// multiple events for the same transfer, and since we are fetching the
		// whole object, we can safely send it once. Let's ignore the transfer
		// if the transaction is a debit. It will be fetch on the other side (
		// the other account's transaction)
		return nil, nil
	}

	transfer, err := p.client.GetTransfer(ctx, transaction.SourceID)
	if err != nil {
		return nil, err
	}

	return translateTransferToPayment(&transfer)
}

func matchTransactionType(transactionType string) models.PaymentType {
	if transactionType == "PI_REV" ||
		transactionType == "PO_REV" ||
		transactionType == "ADHOC" {
		return models.PAYMENT_TYPE_OTHER
	}

	if transactionType == "INT_INTERC" {
		return models.PAYMENT_TYPE_TRANSFER
	}

	if strings.HasPrefix(transactionType, "PI_") {
		return models.PAYMENT_TYPE_PAYIN
	}

	if strings.HasPrefix(transactionType, "PO_") {
		return models.PAYMENT_TYPE_PAYOUT
	}

	return models.PAYMENT_TYPE_OTHER
}
