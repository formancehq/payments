package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Transaction struct {}

func (c *client) GetTransactions(ctx context.Context, page, pageSize int) ([]*Transaction, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")

	// TODO: call PSP to get transactions
	return nil, nil
}
