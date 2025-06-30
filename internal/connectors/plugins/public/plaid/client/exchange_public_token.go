package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/plaid/plaid-go/v34/plaid"
)

type ExchangePublicTokenRequest struct {
	PublicToken string
}

type ExchangePublicTokenResponse struct {
	AccessToken string
	ItemID      string
}

func (c *client) ExchangePublicToken(ctx context.Context, req ExchangePublicTokenRequest) (ExchangePublicTokenResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "exchange_public_token")

	request := plaid.NewItemPublicTokenExchangeRequest(req.PublicToken)
	resp, _, err := c.client.PlaidApi.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(*request).Execute()
	if err != nil {
		return ExchangePublicTokenResponse{}, wrapSDKError(err)
	}

	return ExchangePublicTokenResponse{
		AccessToken: resp.GetAccessToken(),
		ItemID:      resp.GetItemId(),
	}, nil
}
