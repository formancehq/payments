package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/internal/models"
)

// fetchNextPayments paginates /api/v2/user_transactions/ via since_id,
// delegating row-level mapping to mappers.UserTransactionToPSPPayment.
// Trades (type 2) and instant buy/sell (type 36) are surfaced via the
// orders and conversions capabilities respectively; they skip here.
func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	currencies, err := p.getCurrencies(ctx)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	var state paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextPaymentsResponse{}, fmt.Errorf("unmarshal payments state: %w", err)
		}
	}

	limit := effectivePageSize(req.PageSize)
	transactions, err := p.client.GetUserTransactions(ctx, sinceIDFor(state.LastTransactionID), limit)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("fetch payments: %w", err)
	}

	payments := make([]models.PSPPayment, 0, len(transactions))
	lastSeen := state.LastTransactionID
	for _, tx := range transactions {
		if tx.ID > lastSeen {
			lastSeen = tx.ID
		}
		res, err := mappers.UserTransactionToPSPPayment(currencies, tx)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, fmt.Errorf("map payment %d: %w", tx.ID, err)
		}
		if res.DerivativesRow {
			// Spot-only connector — surface this loudly via Error so a
			// derivatives-enabled account is not silently mis-classified.
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

	payload, err := json.Marshal(paymentsState{LastTransactionID: lastSeen})
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("marshal payments state: %w", err)
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		// limit (not req.PageSize) so a zero req.PageSize cannot make
		// HasMore=true on an empty cycle (PR #679 CodeRabbit guard).
		HasMore: len(transactions) == limit,
	}, nil
}

// effectivePageSize guards against the engine passing a non-positive
// PageSize, which would otherwise make HasMore=true when the page is
// empty (the CodeRabbit infinite-loop finding on PR #679 payments.go).
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
