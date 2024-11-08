package client

import (
	"context"
	"time"

	"github.com/get-momo/atlar-v1-go-client/client/transactions"
)

func (c *client) GetV1Transactions(ctx context.Context, token string, pageSize int64) (*transactions.GetV1TransactionsOK, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "list_transactions")

	params := transactions.GetV1TransactionsParams{
		Limit:   &pageSize,
		Context: ctx,
		Token:   &token,
	}

	resp, err := c.client.Transactions.GetV1Transactions(&params)
	return resp, wrapSDKErr(err)
}

func (c *client) GetV1TransactionsID(ctx context.Context, id string) (*transactions.GetV1TransactionsIDOK, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "get_transaction")

	params := transactions.GetV1TransactionsIDParams{
		Context: ctx,
		ID:      id,
	}

	resp, err := c.client.Transactions.GetV1TransactionsID(&params)
	return resp, wrapSDKErr(err)
}
