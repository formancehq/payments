package client

import (
	"context"
	"fmt"
	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"net/http"
)

type Transactions struct {
}

func (c *client) GetTransactions(ctx context.Context, page, pageSize int) ([]Transactions, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_external_accounts")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildEndpoint("v2/transactions"), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("page", fmt.Sprint(page))
	q.Add("per_page", fmt.Sprint(pageSize))
	q.Add("sort_by", "updated_at:asc")
	req.URL.RawQuery = q.Encode()

	errorResponse := qontoErrors{}
	type qontoResponse struct {
		Beneficiaries []Transactions `json:"transactions"`
	}
	successResponse := qontoResponse{}

	_, err = c.httpClient.Do(ctx, req, &successResponse, &errorResponse)

	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get transactions: %v", errorResponse.Error()),
			err,
		)
	}
	return successResponse.Beneficiaries, nil
}
