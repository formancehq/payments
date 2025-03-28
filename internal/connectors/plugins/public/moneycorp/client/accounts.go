package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

type accountsResponse struct {
	Accounts []*Account `json:"data"`
}

type Account struct {
	ID         string `json:"id"`
	Attributes struct {
		AccountName string `json:"accountName"`
	} `json:"attributes"`
}

func (c *client) GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_accounts")

	endpoint := fmt.Sprintf("%s/accounts", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create accounts request: %w", err)
	}

	// TODO generic headers can be set in wrapper
	req.Header.Set("Content-Type", "application/json")

	q := req.URL.Query()
	q.Add("page[size]", strconv.Itoa(pageSize))
	q.Add("page[number]", fmt.Sprint(page))
	q.Add("sortBy", "id.asc")
	req.URL.RawQuery = q.Encode()

	accounts := accountsResponse{Accounts: make([]*Account, 0)}
	var errRes moneycorpErrors
	_, err = c.httpClient.Do(ctx, req, &accounts, &errRes)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get accounts: %v", errRes.Error()),
			err,
		)
	}
	return accounts.Accounts, nil
}
