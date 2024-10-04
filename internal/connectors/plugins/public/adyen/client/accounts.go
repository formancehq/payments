package client

import (
	"context"

	"github.com/adyen/adyen-go-api-library/v7/src/management"
)

func (c *client) GetMerchantAccounts(ctx context.Context, pageNumber, pageSize int32) ([]management.Merchant, error) {
	// TODO(polo): add metrics
	// f := connectors.ClientMetrics(ctx, "adyen", "list_merchant_accounts")
	// now := time.Now()
	// defer f(ctx, now)

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
