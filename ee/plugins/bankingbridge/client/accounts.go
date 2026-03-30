package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/utils/errors"
)

type Account struct {
	Reference    string    `json:"reference"`
	DefaultAsset *string   `json:"defaultAsset,omitempty"`
	Name         *string   `json:"name,omitempty"`
	ImportedAt   time.Time `json:"importedAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (c *client) GetAccounts(ctx context.Context, cursor string, lastImportedAt string, pageSize int) ([]Account, bool, string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_accounts")

	endpoint := fmt.Sprintf("%s/v1/connectors/accounts", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, false, "", fmt.Errorf("failed to create accounts request: %w", err)
	}

	req.URL.RawQuery = RawQuery(req.URL.Query(), pageSize, cursor, lastImportedAt)
	var body struct {
		Cursor struct {
			PageSize int64     `json:"pageSize"`
			Next     string    `json:"next"`
			Previous string    `json:"previous"`
			HasMore  bool      `json:"hasMore"`
			Data     []Account `json:"data"`
		} `json:"cursor"`
	}
	statusCode, err := c.httpClient.Do(ctx, req, &body, nil)
	if err != nil {
		return nil, false, "", errors.NewWrappedError(
			fmt.Errorf("failed to get accounts, status code: %d", statusCode),
			err,
		)
	}
	return body.Cursor.Data, body.Cursor.HasMore, body.Cursor.Next, nil
}
