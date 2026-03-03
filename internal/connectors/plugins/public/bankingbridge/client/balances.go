package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/utils/errors"
)

type Balance struct {
	AccountReference string    `json:"accountReference"`
	Asset            string    `json:"asset"`
	AmountInMinors   int64     `json:"amountInMinors"`
	ReportedAt       time.Time `json:"reportedAt"`

	ImportedAt time.Time `json:"importedAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (c *client) GetAccountBalances(ctx context.Context, cursor string, lastImportedAt string, pageSize int) ([]Balance, bool, string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_account_balances")

	endpoint := fmt.Sprintf("%s/v1/connectors/balances", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, false, "", fmt.Errorf("failed to create balances request: %w", err)
	}

	req.URL.RawQuery = RawQuery(req.URL.Query(), pageSize, cursor, lastImportedAt)
	var body struct {
		Cursor struct {
			PageSize int64     `json:"pageSize"`
			Next     string    `json:"next"`
			Previous string    `json:"previous"`
			HasMore  bool      `json:"hasMore"`
			Data     []Balance `json:"data"`
		} `json:"cursor"`
	}
	statusCode, err := c.httpClient.Do(ctx, req, &body, nil)
	if err != nil {
		return nil, false, "", errors.NewWrappedError(
			fmt.Errorf("failed to get balances, status code: %d", statusCode),
			err,
		)
	}
	return body.Cursor.Data, body.Cursor.HasMore, body.Cursor.Next, nil
}
