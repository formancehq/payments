package client

import (
	"context"
	"time"

	"github.com/formancehq/payments/genericclient"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

func (c *client) ListTransactions(ctx context.Context, page, pageSize int64, updatedAtFrom time.Time) ([]genericclient.Transaction, error) {
	req := c.apiClient.DefaultApi.GetTransactions(metrics.OperationContext(ctx, "list_transactions")).
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
