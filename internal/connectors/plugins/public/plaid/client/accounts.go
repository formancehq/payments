package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/plaid/plaid-go/v34/plaid"
)

func (c *client) ListAccounts(ctx context.Context, accessToken string) (plaid.AccountsGetResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "lsit_accounts")

	request := plaid.NewAccountsGetRequest(accessToken)

	resp, _, err := c.client.PlaidApi.AccountsGet(ctx).AccountsGetRequest(*request).Execute()
	if err != nil {
		return plaid.AccountsGetResponse{}, wrapSDKError(err)
	}

	return resp, nil
}
