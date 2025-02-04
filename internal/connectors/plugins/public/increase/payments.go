package increase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/models"
)

// Using pollingState from accounts.go

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	state, err := p.getPollingState(req.State)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to get polling state: %w", err)
	}

	if state.LastFetch.Add(p.config.PollingPeriod).After(time.Now().UTC()) {
		return models.FetchNextPaymentsResponse{}, nil
	}

	// Get succeeded transactions
	transactions, nextCursor, hasMore, err := p.client.GetTransactions(ctx, state.LastID, int64(req.PageSize))
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to get transactions: %w", err)
	}

	// Get pending transactions
	pendingTransactions, _, _, err := p.client.GetPendingTransactions(ctx, "", int64(req.PageSize))
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to get pending transactions: %w", err)
	}
	transactions = append(transactions, pendingTransactions...)

	// Get declined transactions
	declinedTransactions, _, _, err := p.client.GetDeclinedTransactions(ctx, "", int64(req.PageSize))
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to get declined transactions: %w", err)
	}
	transactions = append(transactions, declinedTransactions...)

	pspPayments := make([]models.PSPPayment, len(transactions))
	for i, tx := range transactions {
		raw, err := json.Marshal(tx)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to marshal transaction: %w", err)
		}

		status := models.PaymentStatusSucceeded
		switch tx.Status {
		case "pending":
			status = models.PaymentStatusPending
		case "declined":
			status = models.PaymentStatusFailed
		}

		pspPayments[i] = models.PSPPayment{
			ID:        tx.ID,
			CreatedAt: tx.CreatedAt,
			Reference: tx.ID,
			Type:      models.PaymentType(tx.Type),
			Status:    status,
			Amount:    tx.Amount,
			Currency:  tx.Currency,
			Raw:       raw,
		}
	}

	newState := pollingState{
		LastID:    nextCursor,
		LastFetch: time.Now().UTC(),
	}
	newStateBytes, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to marshal new state: %w", err)
	}

	return models.FetchNextPaymentsResponse{
		Payments: pspPayments,
		NewState: newStateBytes,
		HasMore:  hasMore,
	}, nil
}
