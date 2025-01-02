package client

import (
	"context"

	"github.com/formancehq/payments/genericclient"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

func (c *client) GetBalances(ctx context.Context, accountID string) (*genericclient.Balances, error) {
	req := c.apiClient.DefaultApi.GetAccountBalances(metrics.OperationContext(ctx, "list_balances"), accountID)

	balances, _, err := req.Execute()
	if err != nil {
		return nil, err
	}

	return balances, nil
}
