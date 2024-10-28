package client

import (
	"context"
	"time"

	"github.com/formancehq/payments/genericclient"
)

func (c *Client) ListBeneficiaries(ctx context.Context, page, pageSize int64, createdAtFrom time.Time) ([]genericclient.Beneficiary, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "list_beneficiaries")

	req := c.apiClient.DefaultApi.
		GetBeneficiaries(ctx).
		Page(page).
		PageSize(pageSize)

	if !createdAtFrom.IsZero() {
		req = req.CreatedAtFrom(createdAtFrom)
	}

	beneficiaries, _, err := req.Execute()
	if err != nil {
		return nil, err
	}

	return beneficiaries, nil
}
