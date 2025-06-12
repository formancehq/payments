package client

import (
	"context"
	"time"

	"github.com/formancehq/payments/genericclient"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

func (c *client) ListBeneficiaries(ctx context.Context, page, pageSize int64, createdAtFrom time.Time) ([]genericclient.Beneficiary, error) {
	req := c.apiClient.DefaultApi.
		GetBeneficiaries(metrics.OperationContext(ctx, "list_beneficiaries")).
		Page(page).
		PageSize(pageSize).
		Sort("createdAt:asc")

	if !createdAtFrom.IsZero() {
		req = req.CreatedAtFrom(createdAtFrom)
	}

	beneficiaries, _, err := req.Execute()
	if err != nil {
		return nil, err
	}

	return beneficiaries, nil
}
