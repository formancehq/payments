package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/get-momo/atlar-v1-go-client/client/accounts"
)

func (c *client) GetV1AccountsID(ctx context.Context, id string) (*accounts.GetV1AccountsIDOK, error) {
	accountsParams := accounts.GetV1AccountsIDParams{
		Context:    metrics.OperationContext(ctx, "get_account"),
		ID:         id,
		HTTPClient: c.httpClient,
	}

	resp, err := c.client.Accounts.GetV1AccountsID(&accountsParams)
	return resp, wrapSDKErr(err)
}

func (c *client) GetV1Accounts(ctx context.Context, token string, pageSize int64) (*accounts.GetV1AccountsOK, error) {
	accountsParams := accounts.GetV1AccountsParams{
		Limit:      &pageSize,
		Context:    metrics.OperationContext(ctx, "list_accounts"),
		Token:      &token,
		HTTPClient: c.httpClient,
	}

	resp, err := c.client.Accounts.GetV1Accounts(&accountsParams)
	return resp, wrapSDKErr(err)
}
