package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/plaid/plaid-go/v34/plaid"
)

type DeleteItemRequest struct {
	AccessToken string
}

func (c *client) DeleteItem(ctx context.Context, req DeleteItemRequest) error {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "delete_item")

	request := plaid.NewItemRemoveRequest(req.AccessToken)
	_, _, err := c.client.PlaidApi.ItemRemove(ctx).ItemRemoveRequest(*request).Execute()
	if err != nil {
		return wrapSDKError(err)
	}

	return nil
}
