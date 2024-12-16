package client

import (
	"context"
	"fmt"
	"net/http"

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
	if err != nil {
		return nil, err
	}

	if raw.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("failed to get merchant accounts: %d", raw.StatusCode)
	}

	return listMerchantsResponse.Data, nil
}
