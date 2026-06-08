package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/genericclient/v3"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

func (c *client) GetBalances(ctx context.Context, accountID string) (*genericclient.Balances, error) {
	ctx = metrics.OperationContext(ctx, "list_balances")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/accounts/%s/balances", c.baseURL, accountID), nil)
	if err != nil {
		return nil, err
	}

	var balances genericclient.Balances
	var errResp genericAPIError
	if _, err = c.httpClient.Do(ctx, req, &balances, &errResp); err != nil {
		return nil, fmt.Errorf("failed to get balances: %w", err)
	}
	return &balances, nil
}
