package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/plaid/plaid-go/v34/plaid"
)

func (c *client) CreateUser(ctx context.Context, userID string) (string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_user")

	request := plaid.NewUserCreateRequest(userID)
	resp, _, err := c.client.PlaidApi.UserCreate(ctx).UserCreateRequest(*request).Execute()
	if err != nil {
		return "", wrapSDKError(err)
	}

	return resp.UserToken, nil
}
