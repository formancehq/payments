package client

import (
	"context"
	"time"

	"github.com/adyen/adyen-go-api-library/v7/src/management"
)

func (c *client) GetMerchantAccounts(ctx context.Context, pageNumber, pageSize int32) ([]management.Merchant, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "list_merchant_accounts")

	listMerchantsResponse, raw, err := c.client.Management().AccountMerchantLevelApi.ListMerchantAccounts(
		ctx,
		c.client.Management().AccountMerchantLevelApi.ListMerchantAccountsInput().PageNumber(pageNumber).PageSize(pageSize),
	)
	err = c.wrapSDKError(err, raw.StatusCode)
	if err != nil {
		return nil, err
	}
	return listMerchantsResponse.Data, nil
}
