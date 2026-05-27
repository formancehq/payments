package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/internal/models"
)

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
	transactions, err := p.client.GetUserTransactions(ctx, sinceIDFor(state.LastTransactionID), limit)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to poll user_transactions: %w", err)
	}

	payments := make([]models.PSPPayment, 0, len(transactions))
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

	payload, err := json.Marshal(state)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to marshal payments state: %w", err)
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  len(transactions) == limit,
	}, nil
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
