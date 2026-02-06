package client

import (
	"context"

	"github.com/formancehq/payments/pkg/connector/metrics"
	"github.com/get-momo/atlar-v1-go-client/client/transactions"
)

func (c *client) GetV1Transactions(ctx context.Context, token string, pageSize int64) (*transactions.GetV1TransactionsOK, error) {
	params := transactions.GetV1TransactionsParams{
		Limit:      &pageSize,
		Context:    metrics.OperationContext(ctx, "list_transactions"),
		Token:      &token,
		HTTPClient: c.httpClient,
	}

	resp, err := c.client.Transactions.GetV1Transactions(&params)
	return resp, wrapSDKErr(err, nil)
}

func (c *client) GetV1TransactionsID(ctx context.Context, id string) (*transactions.GetV1TransactionsIDOK, error) {
	params := transactions.GetV1TransactionsIDParams{
		Context:    metrics.OperationContext(ctx, "get_transaction"),
		ID:         id,
		HTTPClient: c.httpClient,
	}

	resp, err := c.client.Transactions.GetV1TransactionsID(&params)
	return resp, wrapSDKErr(err, &transactions.GetV1TransactionsIDNotFound{})
}
