package client

import (
	"context"
	"time"

	"github.com/get-momo/atlar-v1-go-client/client/accounts"
)

func (c *client) GetV1AccountsID(ctx context.Context, id string) (*accounts.GetV1AccountsIDOK, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "get_account")

	accountsParams := accounts.GetV1AccountsIDParams{
		Context: ctx,
		ID:      id,
	}

	return c.client.Accounts.GetV1AccountsID(&accountsParams)
}

func (c *client) GetV1Accounts(ctx context.Context, token string, pageSize int64) (*accounts.GetV1AccountsOK, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "list_accounts")

	accountsParams := accounts.GetV1AccountsParams{
		Limit:   &pageSize,
		Context: ctx,
		Token:   &token,
	}

	return c.client.Accounts.GetV1Accounts(&accountsParams)
}
