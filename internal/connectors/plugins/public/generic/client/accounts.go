package client

import (
	"context"
	"time"

	"github.com/formancehq/payments/genericclient"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

func (c *client) ListAccounts(ctx context.Context, page, pageSize int64, createdAtFrom time.Time) ([]genericclient.Account, error) {
	req := c.apiClient.DefaultApi.
		GetAccounts(metrics.OperationContext(ctx, "list_accounts")).
		Page(page).
		PageSize(pageSize)

	if !createdAtFrom.IsZero() {
		req = req.CreatedAtFrom(createdAtFrom)
	}

	accounts, _, err := req.Execute()
	if err != nil {
		return nil, err
	}

	return accounts, nil
}
