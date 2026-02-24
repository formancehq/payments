package client

import (
	"context"

	"github.com/formancehq/payments/pkg/connector/metrics"
	"github.com/get-momo/atlar-v1-go-client/client/external_accounts"
)

func (c *client) GetV1ExternalAccountsID(ctx context.Context, externalAccountID string) (*external_accounts.GetV1ExternalAccountsIDOK, error) {
	getExternalAccountParams := external_accounts.GetV1ExternalAccountsIDParams{
		Context:    metrics.OperationContext(ctx, "get_external_account"),
		ID:         externalAccountID,
		HTTPClient: c.httpClient,
	}

	externalAccountResponse, err := c.client.ExternalAccounts.GetV1ExternalAccountsID(&getExternalAccountParams)
	return externalAccountResponse, wrapSDKErr(err, &external_accounts.GetV1ExternalAccountsIDNotFound{})
}

func (c *client) GetV1ExternalAccounts(ctx context.Context, token string, pageSize int64) (*external_accounts.GetV1ExternalAccountsOK, error) {
	externalAccountsParams := external_accounts.GetV1ExternalAccountsParams{
		Limit:      &pageSize,
		Context:    metrics.OperationContext(ctx, "list_external_accounts"),
		Token:      &token,
		HTTPClient: c.httpClient,
	}

	resp, err := c.client.ExternalAccounts.GetV1ExternalAccounts(&externalAccountsParams)
	return resp, wrapSDKErr(err, nil)
}
