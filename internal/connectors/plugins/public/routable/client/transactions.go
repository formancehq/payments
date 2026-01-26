package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Transaction struct {
	ID                  string `json:"id"`
	CreatedAt           string `json:"created_at"`
	Amount              string `json:"amount"`
	CurrencyCode        string `json:"currency_code"`
	Status              string `json:"status"`
	Type                string `json:"type"`
	WithdrawFromAccount struct {
		ID string `json:"id"`
	} `json:"withdraw_from_account"`
}

type payablesList struct {
	Object  string        `json:"object"`
	Results []Transaction `json:"results"`
	Links   struct {
		Self string  `json:"self"`
		Next *string `json:"next"`
		Prev *string `json:"prev"`
	} `json:"links"`
}

func (c *client) buildPayablesPath(page, pageSize int, withdrawFromAccount string) string {
	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}
	if withdrawFromAccount != "" {
		q.Add("withdraw_from_account", withdrawFromAccount)
	}
	path := "/v1/payables"
	if len(q) > 0 {
		path = fmt.Sprintf("%s?%s", path, q.Encode())
	}
	return path
}

func (c *client) GetTransactions(ctx context.Context, page, pageSize int) ([]*Transaction, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_payables")
	path := c.buildPayablesPath(page, pageSize, "")
	return c.fetchPayablesPage(ctx, path)
}

func (c *client) GetTransactionsByAccount(ctx context.Context, page, pageSize int, accountID string) ([]*Transaction, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_payables_by_account")
	path := c.buildPayablesPath(page, pageSize, accountID)
	return c.fetchPayablesPage(ctx, path)
}

func (c *client) fetchPayablesPage(ctx context.Context, path string) ([]*Transaction, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var out payablesList
	if _, err := c.httpClient.Do(ctx, req, &out, &out); err != nil {
		return nil, err
	}
	res := make([]*Transaction, 0, len(out.Results))
	for i := range out.Results {
		t := out.Results[i]
		res = append(res, &t)
	}
	return res, nil
}
