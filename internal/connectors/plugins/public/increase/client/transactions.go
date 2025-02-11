package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Transaction struct {
	ID          string `json:"id"`
	AccountID   string `json:"account_id"`
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	CreatedAt   string `json:"created_at"`
	Date        string `json:"date"`
	Description string `json:"description"`
	RouteID     string `json:"route_id"`
	RouteType   string `json:"route_type"`
	Source      struct {
		DestinationAccountID string `json:"destination_account_id"`
		SourceAccountID      string `json:"source_account_id"`
		TransactionID        string `json:"transaction_id"`
	} `json:"source"`
}

func (c *client) GetTransactions(ctx context.Context, pageSize int, cursor string) ([]*Transaction, string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")

	req, err := c.newRequest(ctx, http.MethodGet, "transactions", http.NoBody)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create transactions request: %w", err)
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(pageSize))
	if cursor != "" {
		q.Add("cursor", cursor)
	}
	req.URL.RawQuery = q.Encode()

	var res responseWrapper[[]*Transaction]
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get transactions: %w %w", err, errRes.Error())
	}

	return res.Data, res.NextCursor, nil
}

func (c *client) GetPendingTransactions(ctx context.Context, pageSize int, cursor string) ([]*Transaction, string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_pending_transactions")

	req, err := c.newRequest(ctx, http.MethodGet, "pending_transactions", http.NoBody)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create pending transactions request: %w", err)
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(pageSize))
	if cursor != "" {
		q.Add("cursor", cursor)
	}
	req.URL.RawQuery = q.Encode()

	var res responseWrapper[[]*Transaction]
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get pending transactions: %w %w", err, errRes.Error())
	}

	return res.Data, res.NextCursor, nil
}

func (c *client) GetDeclinedTransactions(ctx context.Context, pageSize int, cursor string) ([]*Transaction, string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_declined_transactions")

	req, err := c.newRequest(ctx, http.MethodGet, "declined_transactions", http.NoBody)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create declined transactions request: %w", err)
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(pageSize))
	if cursor != "" {
		q.Add("cursor", cursor)
	}
	req.URL.RawQuery = q.Encode()

	var res responseWrapper[[]*Transaction]
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get declined transactions: %w %w", err, errRes.Error())
	}

	return res.Data, res.NextCursor, nil
}
