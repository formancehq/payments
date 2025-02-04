package client

import (
	"context"
)

type Balance struct {
	Available int64  `json:"available"`
	Currency  string `json:"currency"`
}

func (c *client) GetAccountBalances(ctx context.Context, accountID string) ([]*Balance, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_account_balance")

	endpoint := fmt.Sprintf("/accounts/%s/balance", accountID)
	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Available int64  `json:"available"`
		Currency  string `json:"currency"`
	}
	if err := c.do(req, &response); err != nil {
		return nil, err
	}

	return []*Balance{{
		Available: response.Available,
		Currency:  response.Currency,
	}}, nil
}
