package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/formancehq/payments/ce/plugins/generic/client/generated"
	"github.com/formancehq/payments/pkg/domain/metrics"
)

func (c *client) ListTransactions(ctx context.Context, page, pageSize int64, updatedAtFrom time.Time) ([]genericclient.Transaction, error) {
	ctx = metrics.OperationContext(ctx, "list_transactions")

	u, err := url.Parse(fmt.Sprintf("%s/transactions", c.baseURL))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("page", strconv.FormatInt(page, 10))
	q.Set("pageSize", strconv.FormatInt(pageSize, 10))
	q.Set("sort", "updatedAt:asc")
	if !updatedAtFrom.IsZero() {
		q.Set("updatedAtFrom", updatedAtFrom.UTC().Format(time.RFC3339))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	var transactions []genericclient.Transaction
	var errResp genericAPIError
	if _, err = c.httpClient.Do(ctx, req, &transactions, &errResp); err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	return transactions, nil
}
