package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/pkg/connector/metrics"
	"github.com/formancehq/payments/pkg/connector"
)

type Balance struct {
	ID        string      `json:"id"`
	AccountID string      `json:"account_id"`
	Currency  string      `json:"currency"`
	Amount    json.Number `json:"amount"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

func (c *client) GetBalances(ctx context.Context, page int, pageSize int) ([]*Balance, int, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_balances")

	if err := c.ensureLogin(ctx); err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.buildEndpoint("v2/balances/find"), http.NoBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("per_page", fmt.Sprint(pageSize))
	q.Add("page", fmt.Sprint(page))
	q.Add("order", "created_at")
	q.Add("order_asc_desc", "asc")
	req.URL.RawQuery = q.Encode()

	req.Header.Add("Accept", "application/json")

	//nolint:tagliatelle // allow for client code
	type response struct {
		Balances   []*Balance `json:"balances"`
		Pagination struct {
			NextPage int `json:"next_page"`
		} `json:"pagination"`
	}

	res := response{Balances: make([]*Balance, 0)}
	var errRes currencyCloudError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, 0, connector.NewWrappedError(
			fmt.Errorf("failed to get balances: %v", errRes.Error()),
			err,
		)
	}
	return res.Balances, res.Pagination.NextPage, nil
}
