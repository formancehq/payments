package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

const (
	createdAtAfterQueryParam = "created_at.after"
)

type Transaction struct {
	ID          string      `json:"id"`
	AccountID   string      `json:"account_id"`
	Amount      json.Number `json:"amount"`
	Currency    string      `json:"currency"`
	CreatedAt   string      `json:"created_at"`
	Date        string      `json:"date"`
	Description string      `json:"description"`
	RouteID     string      `json:"route_id"`
	RouteType   string      `json:"route_type"`
	Type        string      `json:"type"`
}

func (c *client) GetTransactions(ctx context.Context, pageSize int, lastCreatedAtAfter time.Time, nextCursor string) ([]*Transaction, string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")

	req, err := c.newRequest(ctx, http.MethodGet, "transactions", http.NoBody)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create transactions request: %w", err)
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(pageSize))
	if nextCursor != "" {
		q.Add("cursor", nextCursor)
	}
	if !lastCreatedAtAfter.IsZero() && nextCursor == "" {
		q.Add(createdAtAfterQueryParam, lastCreatedAtAfter.Format(time.RFC3339))
	}
	req.URL.RawQuery = q.Encode()

	var res ResponseWrapper[[]*Transaction]
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get transactions: %w %w", err, errRes.Error())
	}

	return res.Data, res.NextCursor, nil
}

func (c *client) GetTransaction(ctx context.Context, transactionID string) (*Transaction, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_transaction")

	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("transactions/%s", transactionID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactions request: %w", err)
	}

	var res Transaction
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w %w", err, errRes.Error())
	}

	return &res, nil
}

func (c *client) GetPendingTransactions(ctx context.Context, pageSize int, lastCreatedAtAfter time.Time, nextCursor string) ([]*Transaction, string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_pending_transactions")

	req, err := c.newRequest(ctx, http.MethodGet, "pending_transactions", http.NoBody)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create pending transactions request: %w", err)
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(pageSize))
	if nextCursor != "" {
		q.Add("cursor", nextCursor)
	}
	if !lastCreatedAtAfter.IsZero() && nextCursor == "" {
		q.Add(createdAtAfterQueryParam, lastCreatedAtAfter.Format(time.RFC3339))
	}
	req.URL.RawQuery = q.Encode()

	var res ResponseWrapper[[]*Transaction]
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get pending transactions: %w %w", err, errRes.Error())
	}

	return res.Data, res.NextCursor, nil
}

func (c *client) GetPendingTransaction(ctx context.Context, transactionID string) (*Transaction, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_pending_transaction")

	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("pending_transactions/%s", transactionID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create pending transactions request: %w", err)
	}

	var res Transaction
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending transaction: %w %w", err, errRes.Error())
	}

	return &res, nil
}

func (c *client) GetDeclinedTransactions(ctx context.Context, pageSize int, lastCreatedAtAfter time.Time, nextCursor string) ([]*Transaction, string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_declined_transactions")

	req, err := c.newRequest(ctx, http.MethodGet, "declined_transactions", http.NoBody)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create declined transactions request: %w", err)
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(pageSize))
	if nextCursor != "" {
		q.Add("cursor", nextCursor)
	}
	if !lastCreatedAtAfter.IsZero() && nextCursor == "" {
		q.Add(createdAtAfterQueryParam, lastCreatedAtAfter.Format(time.RFC3339))
	}
	req.URL.RawQuery = q.Encode()

	var res ResponseWrapper[[]*Transaction]
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get declined transactions: %w %w", err, errRes.Error())
	}

	return res.Data, res.NextCursor, nil
}

func (c *client) GetDeclinedTransaction(ctx context.Context, transactionID string) (*Transaction, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_declined_transaction")

	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("declined_transactions/%s", transactionID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create declined transaction request: %w", err)
	}

	var res Transaction
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get declined transaction: %w %w", err, errRes.Error())
	}

	return &res, nil
}
