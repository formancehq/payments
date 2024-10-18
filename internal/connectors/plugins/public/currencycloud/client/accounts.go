package client

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Account struct {
	ID          string    `json:"id"`
	AccountName string    `json:"account_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (c *client) GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, int, error) {
	// TODO(polo): metrics
	// f := connectors.ClientMetrics(ctx, "currencycloud", "list_accounts")
	// now := time.Now()
	// defer f(ctx, now)

	if err := c.ensureLogin(ctx); err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.buildEndpoint("v2/accounts/find"), http.NoBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("per_page", fmt.Sprint(pageSize))
	q.Add("page", fmt.Sprint(page))
	q.Add("order", "updated_at")
	q.Add("order_asc_desc", "asc")
	req.URL.RawQuery = q.Encode()

	req.Header.Add("Accept", "application/json")

	//nolint:tagliatelle // allow for client code
	type response struct {
		Accounts   []*Account `json:"accounts"`
		Pagination struct {
			NextPage int `json:"next_page"`
		} `json:"pagination"`
	}

	res := response{Accounts: make([]*Account, 0)}
	var errRes currencyCloudError
	_, err = c.httpClient.Do(req, &res, &errRes)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get accounts: %w, %w", err, errRes.Error())
	}
	return res.Accounts, res.Pagination.NextPage, nil
}