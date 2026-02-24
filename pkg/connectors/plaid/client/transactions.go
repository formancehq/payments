package client

import (
	"context"

	"github.com/formancehq/payments/pkg/connector/metrics"
	"github.com/plaid/plaid-go/v34/plaid"
)

func (c *client) ListTransactions(ctx context.Context, accessToken string, cursor string, pageSize int) (plaid.TransactionsSyncResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")

	request := plaid.NewTransactionsSyncRequest(accessToken)
	if cursor != "" {
		request.SetCursor(cursor)
	}
	if pageSize > 0 {
		request.SetCount(int32(pageSize))
	}

	resp, _, err := c.client.PlaidApi.TransactionsSync(ctx).TransactionsSyncRequest(*request).Execute()
	if err != nil {
		return plaid.TransactionsSyncResponse{}, wrapSDKError(err)
	}

	return resp, nil
}
