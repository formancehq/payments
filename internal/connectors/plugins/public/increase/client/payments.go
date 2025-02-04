package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/pkg/metrics"
)

type Transaction struct {
	ID            string    `json:"id"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	AccountID     string    `json:"account_id"`
	RouteID       string    `json:"route_id"`
	RouteType     string    `json:"route_type"`
	Description   string    `json:"description"`
}

func (c *client) GetTransactions(ctx context.Context, lastID string, pageSize int64) ([]*Transaction, string, bool, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")

	endpoint := fmt.Sprintf("/transactions?limit=%d&status=succeeded", pageSize)
	if lastID != "" {
		endpoint += "&cursor=" + lastID
	}

	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", false, err
	}

	var response struct {
		Data     []*Transaction `json:"data"`
		NextPage string         `json:"next_page"`
	}
	if err := c.do(req, &response); err != nil {
		return nil, "", false, err
	}

	return response.Data, response.NextPage, response.NextPage != "", nil
}

func (c *client) GetPendingTransactions(ctx context.Context, lastID string, pageSize int64) ([]*Transaction, string, bool, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_pending_transactions")

	endpoint := fmt.Sprintf("/transactions?limit=%d&status=pending", pageSize)
	if lastID != "" {
		endpoint += "&cursor=" + lastID
	}

	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", false, err
	}

	var response struct {
		Data     []*Transaction `json:"data"`
		NextPage string         `json:"next_page"`
	}
	if err := c.do(req, &response); err != nil {
		return nil, "", false, err
	}

	return response.Data, response.NextPage, response.NextPage != "", nil
}

func (c *client) GetDeclinedTransactions(ctx context.Context, lastID string, pageSize int64) ([]*Transaction, string, bool, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_declined_transactions")

	endpoint := fmt.Sprintf("/transactions?limit=%d&status=declined", pageSize)
	if lastID != "" {
		endpoint += "&cursor=" + lastID
	}

	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", false, err
	}

	var response struct {
		Data     []*Transaction `json:"data"`
		NextPage string         `json:"next_page"`
	}
	if err := c.do(req, &response); err != nil {
		return nil, "", false, err
	}

	return response.Data, response.NextPage, response.NextPage != "", nil
}
