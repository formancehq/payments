package client

import (
	"context"
	"time"

	"github.com/formancehq/payments/genericclient"
)

func (c *Client) GetBalances(ctx context.Context, accountID string) (*genericclient.Balances, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "list_balances")

	req := c.apiClient.DefaultApi.GetAccountBalances(ctx, accountID)

	balances, _, err := req.Execute()
	if err != nil {
		return nil, err
	}

	return balances, nil
}
