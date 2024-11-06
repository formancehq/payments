package client

import (
	"context"
	"time"

	"github.com/get-momo/atlar-v1-go-client/client/external_accounts"
)

func (c *client) GetV1ExternalAccountsID(ctx context.Context, externalAccountID string) (*external_accounts.GetV1ExternalAccountsIDOK, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "get_external_account")

	getExternalAccountParams := external_accounts.GetV1ExternalAccountsIDParams{
		Context: ctx,
		ID:      externalAccountID,
	}

	externalAccountResponse, err := c.client.ExternalAccounts.GetV1ExternalAccountsID(&getExternalAccountParams)
	if err != nil {
		return nil, err
	}

	return externalAccountResponse, nil
}

func (c *client) GetV1ExternalAccounts(ctx context.Context, token string, pageSize int64) (*external_accounts.GetV1ExternalAccountsOK, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "list_external_accounts")

	externalAccountsParams := external_accounts.GetV1ExternalAccountsParams{
		Limit:   &pageSize,
		Context: ctx,
		Token:   &token,
	}

	return c.client.ExternalAccounts.GetV1ExternalAccounts(&externalAccountsParams)
}
