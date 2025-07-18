package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/plaid/plaid-go/v34/plaid"
)

func (c *client) DeleteUser(ctx context.Context, userToken string) error {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "delete_user")

	request := plaid.NewUserRemoveRequest(userToken)
	_, _, err := c.client.PlaidApi.UserRemove(ctx).UserRemoveRequest(*request).Execute()
	if err != nil {
		return wrapSDKError(err)
	}

	return nil
}
