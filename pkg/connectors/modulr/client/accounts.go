package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/formancehq/payments/pkg/connector/metrics"
	"github.com/formancehq/payments/pkg/connector"
)

//nolint:tagliatelle // allow for clients
type Account struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Balance     string `json:"balance"`
	Currency    string `json:"currency"`
	CustomerID  string `json:"customerId"`
	Identifiers []struct {
		AccountNumber string `json:"accountNumber"`
		SortCode      string `json:"sortCode"`
		Type          string `json:"type"`
	} `json:"identifiers"`
	DirectDebit bool   `json:"directDebit"`
	CreatedDate string `json:"createdDate"`
}

func (c *client) GetAccounts(ctx context.Context, page, pageSize int, fromCreatedAt time.Time) ([]Account, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_accounts")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildEndpoint("accounts"), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create accounts request: %w", err)
	}

	q := req.URL.Query()
	q.Add("page", strconv.Itoa(page))
	q.Add("size", strconv.Itoa(pageSize))
	q.Add("sortField", "createdDate")
	q.Add("sortOrder", "asc")
	if !fromCreatedAt.IsZero() {
		q.Add("fromCreatedDate", fromCreatedAt.Format("2006-01-02T15:04:05-0700"))
	}
	req.URL.RawQuery = q.Encode()

	var res responseWrapper[[]Account]
	var errRes modulrErrors
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, connector.NewWrappedError(
			fmt.Errorf("failed to get accounts: %v", errRes.Error()),
			err,
		)
	}
	return res.Content, nil
}

func (c *client) GetAccount(ctx context.Context, accountID string) (*Account, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_account")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildEndpoint("accounts/%s", accountID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create accounts request: %w", err)
	}

	var res Account
	var errRes modulrErrors
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, connector.NewWrappedError(
			fmt.Errorf("failed to get account: %v", errRes.Error()),
			err,
		)
	}
	return &res, nil
}
