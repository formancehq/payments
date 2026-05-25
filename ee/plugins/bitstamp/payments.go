package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/internal/models"
)

const (
	withdrawalRequestsPageSize = 1000
	cryptoTransactionsPageSize = 1000
)

// fetchNextPayments unions three Bitstamp endpoints into a single
// PSPPayment stream — user_transactions/ (settled history),
// crypto-transactions/ (on-chain, Main-account only), and
// withdrawal-requests/ (fiat lifecycle). See MAPPINGS §4.3.
//
// Any source error short-circuits the cycle: the engine treats the
// returned error as a total cycle failure, so partial success cannot
// be propagated. The next tick retries from the unchanged watermark.
func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var state paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to unmarshal payments state: %w", err)
		}
	}

	currencies, err := p.getCurrencies(ctx)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	limit := effectivePageSize(req.PageSize)
	payments := make([]models.PSPPayment, 0, limit)

	utPayments, utHasMore, err := p.pollUserTransactions(ctx, currencies, &state.UserTransactions, limit)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to poll user_transactions: %w", err)
	}
	payments = append(payments, utPayments...)

	ctPayments, ctHasMore, err := p.pollCryptoTransactions(ctx, currencies, &state.CryptoTransactions)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to poll crypto_transactions: %w", err)
	}
	payments = append(payments, ctPayments...)

	wrPayments, wrHasMore, err := p.pollWithdrawalRequests(ctx, currencies, &state.WithdrawalRequests)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to poll withdrawal_requests: %w", err)
	}
	payments = append(payments, wrPayments...)

	// Deterministic order on the merged batch by (CreatedAt, Reference).
	sort.SliceStable(payments, func(i, j int) bool {
		if !payments[i].CreatedAt.Equal(payments[j].CreatedAt) {
			return payments[i].CreatedAt.Before(payments[j].CreatedAt)
		}
		return payments[i].Reference < payments[j].Reference
	})

	payload, err := json.Marshal(state)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to marshal payments state: %w", err)
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  utHasMore || ctHasMore || wrHasMore,
	}, nil
}

func (p *Plugin) pollUserTransactions(
	ctx context.Context,
	currencies map[string]int,
	state *userTransactionsState,
	limit int,
) (payments []models.PSPPayment, hasMore bool, err error) {
	transactions, err := p.client.GetUserTransactions(ctx, sinceIDFor(state.LastTransactionID), limit)
	if err != nil {
		return nil, false, err
	}
	payments = make([]models.PSPPayment, 0, len(transactions))
	for _, tx := range transactions {
		state.LastTransactionID = advanceInt64Cursor(state.LastTransactionID, tx.ID)
		res, mapErr := mappers.UserTransactionToPSPPayment(currencies, tx)
		if mapErr != nil {
			p.logger.WithField("txID", tx.ID).Errorf("failed to map user_transaction: %v", mapErr)
			continue
		}
		if res.DerivativesRow {
			p.logger.WithField("txID", tx.ID).Errorf("skipping derivatives-marked row on spot-only connector")
			continue
		}
		if res.Skip || res.Payment == nil {
			continue
		}
		if res.UnknownType {
			p.logger.WithField("txID", tx.ID).WithField("txType", tx.Type).
				Infof("emitting payment with PAYMENT_TYPE_OTHER for previously-unseen Bitstamp tx type")
		}
		payments = append(payments, *res.Payment)
	}
	return payments, len(transactions) == limit, nil
}

// pollCryptoTransactions polls /crypto-transactions/ for on-chain
// deposit, withdrawal, and Ripple IOU activity.
func (p *Plugin) pollCryptoTransactions(
	ctx context.Context,
	currencies map[string]int,
	state *cryptoTransactionsState,
) (payments []models.PSPPayment, hasMore bool, err error) {
	opts := client.CryptoTransactionsOptions{
		Limit:       cryptoTransactionsPageSize,
		IncludeIOUs: true,
	}
	resp, err := p.client.GetCryptoTransactions(ctx, opts)
	if err != nil {
		return nil, false, err
	}
	payments = make([]models.PSPPayment, 0, len(resp.Deposits)+len(resp.Withdrawals)+len(resp.RippleIOUTransactions))
	seenDepositTs := make([]int64, 0, len(resp.Deposits))
	for _, d := range resp.Deposits {
		seenDepositTs = append(seenDepositTs, d.Datetime)
		mapped, mapErr := mappers.CryptoDepositToPSPPayment(currencies, d)
		if mapErr != nil {
			p.logger.WithField("deposit_id", d.ID).Errorf("map crypto deposit: %v", mapErr)
			continue
		}
		if mapped == nil {
			p.logger.WithField("deposit_id", d.ID).WithField("currency", d.Currency).
				Infof("skipping crypto deposit with unsupported currency")
			continue
		}
		payments = append(payments, *mapped)
	}
	state.DepositsSinceTs = advanceInt64Cursor(state.DepositsSinceTs, maxInt64(seenDepositTs))

	seenWdTs := make([]int64, 0, len(resp.Withdrawals))
	for _, w := range resp.Withdrawals {
		seenWdTs = append(seenWdTs, w.Datetime)
		mapped, mapErr := mappers.CryptoWithdrawalToPSPPayment(currencies, w)
		if mapErr != nil {
			p.logger.WithField("withdrawal_txid", w.TxID).Errorf("map crypto withdrawal: %v", mapErr)
			continue
		}
		if mapped == nil {
			continue
		}
		payments = append(payments, *mapped)
	}
	state.WithdrawalsSinceTs = advanceInt64Cursor(state.WithdrawalsSinceTs, maxInt64(seenWdTs))

	seenIouTs := make([]int64, 0, len(resp.RippleIOUTransactions))
	for _, r := range resp.RippleIOUTransactions {
		seenIouTs = append(seenIouTs, r.Datetime)
		mapped, mapErr := mappers.RippleIOUToPSPPayment(currencies, r)
		if mapErr != nil {
			p.logger.WithField("iou_txid", r.TxID).Errorf("map ripple IOU: %v", mapErr)
			continue
		}
		if mapped == nil {
			continue
		}
		payments = append(payments, *mapped)
	}
	state.RipplesSinceTs = advanceInt64Cursor(state.RipplesSinceTs, maxInt64(seenIouTs))

	// HasMore is true when any bucket returned a full page — the
	// endpoint exposes per-call limit but not a wire-level "more"
	// flag, so we conservatively keep paging on a full response.
	hasMore = len(resp.Deposits) == cryptoTransactionsPageSize ||
		len(resp.Withdrawals) == cryptoTransactionsPageSize ||
		len(resp.RippleIOUTransactions) == cryptoTransactionsPageSize
	return payments, hasMore, nil
}

// pollWithdrawalRequests polls /withdrawal-requests/ for the fiat
// withdrawal lifecycle.
//
// Bitstamp requires BOTH limit AND offset (the client guards
// against the missing-pair error). We always ask for the page
// starting at offset 0 + sort by id descending (clients receive
// rows newest-first); the orchestrator filters to id > LastID
// after the call.
func (p *Plugin) pollWithdrawalRequests(
	ctx context.Context,
	currencies map[string]int,
	state *withdrawalRequestsState,
) (payments []models.PSPPayment, hasMore bool, err error) {
	rows, err := p.client.GetWithdrawalRequests(ctx, withdrawalRequestsPageSize, 0)
	if err != nil {
		return nil, false, err
	}
	payments = make([]models.PSPPayment, 0, len(rows))
	for _, w := range rows {
		if w.ID <= state.LastID {
			continue
		}
		state.LastID = advanceInt64Cursor(state.LastID, w.ID)
		mapped, mapErr := mappers.WithdrawalRequestToPSPPayment(currencies, w)
		if mapErr != nil {
			p.logger.WithField("withdrawal_id", w.ID).Errorf("map withdrawal request: %v", mapErr)
			continue
		}
		if mapped == nil {
			p.logger.WithField("withdrawal_id", w.ID).WithField("currency", w.Currency).
				Infof("skipping withdrawal request with unsupported currency")
			continue
		}
		payments = append(payments, *mapped)
	}
	// HasMore is true when the page came back full AND advanced the
	// cursor — both conditions matter because a full page of rows
	// already below the watermark cannot make progress.
	hasMore = len(rows) == withdrawalRequestsPageSize && len(payments) > 0
	return payments, hasMore, nil
}

// effectivePageSize guards against the engine passing a non-positive
// PageSize, which would otherwise make HasMore=true when the page is
// empty.
func effectivePageSize(requested int) int {
	if requested <= 0 {
		return PAGE_SIZE
	}
	return requested
}

// sinceIDFor returns a *int64 suitable for the client's since_id
// argument: nil on a cold start (state.LastTransactionID == 0) so the
// initial cycle walks from the earliest available row.
func sinceIDFor(lastID int64) *int64 {
	if lastID <= 0 {
		return nil
	}
	return &lastID
}
