package increase

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
)

type paymentsState struct {
	LastID   string          `json:"last_id"`
	Timeline json.RawMessage `json:"timeline"`
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var state paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to unmarshal state: %w", err)
		}
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

	newState := paymentsState{
		LastID: nextCursor,
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
