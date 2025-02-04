package client

import (
	"context"
	"fmt"

	"github.com/Increase/increase-go"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

func (c *client) GetTransactions(ctx context.Context, page, pageSize int) ([]*Transaction, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")

	params := increase.TransactionListParams{
		Limit: increase.F(int64(pageSize)),
	}
	if page > 0 {
		params.Cursor = increase.F(fmt.Sprintf("%d", page*pageSize))
	}

	resp, err := c.increaseClient.Transactions.List(ctx, params)
	if err != nil {
		return nil, err
	}

	transactions := make([]*Transaction, len(resp.Data))
	for i, tx := range resp.Data {
		transactions[i] = &Transaction{
			ID:          string(tx.ID),
			Type:        string(tx.Type),
			Status:      string(tx.Type),
			Amount:      tx.Amount,
			Currency:    string(tx.Currency),
			Description: tx.Description,
			CreatedAt:   tx.CreatedAt.String(),
		}
	}

	return transactions, nil
}
