package client

import (
	"context"
	"time"

	"github.com/formancehq/payments/genericclient"
)

func (c *client) ListTransactions(ctx context.Context, page, pageSize int64, updatedAtFrom time.Time) ([]genericclient.Transaction, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "list_transactions")

	req := c.apiClient.DefaultApi.GetTransactions(ctx).
		Page(page).
		PageSize(pageSize)

	if !updatedAtFrom.IsZero() {
		req = req.UpdatedAtFrom(updatedAtFrom)
	}

	transactions, _, err := req.Execute()
	if err != nil {
		return nil, err
	}

	return transactions, nil
}
