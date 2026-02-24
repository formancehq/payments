package client

import (
	"context"

	"github.com/adyen/adyen-go-api-library/v7/src/management"
	"github.com/formancehq/payments/pkg/connector/metrics"
)

func (c *client) GetMerchantAccounts(ctx context.Context, pageNumber, pageSize int32) ([]management.Merchant, error) {
	listMerchantsResponse, raw, err := c.client.Management().AccountMerchantLevelApi.ListMerchantAccounts(
		metrics.OperationContext(ctx, "list_merchant_accounts"),
		c.client.Management().AccountMerchantLevelApi.ListMerchantAccountsInput().PageNumber(pageNumber).PageSize(pageSize),
	)
	err = c.wrapSDKError(err, raw.StatusCode)
	if err != nil {
		return nil, err
	}
	return listMerchantsResponse.Data, nil
}
