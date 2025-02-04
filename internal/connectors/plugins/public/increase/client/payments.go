package client

import (
	"context"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/increase/increase-go"
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

func mapTransaction(t *increase.Transaction) *Transaction {
	return &Transaction{
		ID:          t.ID,
		Amount:      t.Amount,
		Currency:    string(t.Currency),
		Type:        string(t.Type),
		Status:      string(t.Status),
		CreatedAt:   t.CreatedAt,
		AccountID:   t.AccountID,
		RouteID:     t.RouteID,
		RouteType:   string(t.RouteType),
		Description: t.Description,
	}
}

func (c *client) GetTransactions(ctx context.Context, lastID string, pageSize int64) ([]*Transaction, string, bool, error) {
	ctx = context.WithValue(ctx, api.MetricOperationContextKey, "list_transactions")

	params := &increase.TransactionListParams{
		Limit:  increase.F(int32(pageSize)),
		Status: increase.F(increase.TransactionListParamsStatusSucceeded),
	}
	if lastID != "" {
		params.Cursor = increase.F(lastID)
	}

	resp, err := c.sdk.Transactions.List(ctx, params)
	if err != nil {
		return nil, "", false, err
	}

	transactions := make([]*Transaction, len(resp.Data))
	for i, t := range resp.Data {
		transactions[i] = mapTransaction(t)
	}

	return transactions, resp.NextCursor, resp.HasMore, nil
}

func (c *client) GetPendingTransactions(ctx context.Context, lastID string, pageSize int64) ([]*Transaction, string, bool, error) {
	ctx = context.WithValue(ctx, api.MetricOperationContextKey, "list_pending_transactions")

	params := &increase.TransactionListParams{
		Limit:  increase.F(int32(pageSize)),
		Status: increase.F(increase.TransactionListParamsStatusPending),
	}
	if lastID != "" {
		params.Cursor = increase.F(lastID)
	}

	resp, err := c.sdk.Transactions.List(ctx, params)
	if err != nil {
		return nil, "", false, err
	}

	transactions := make([]*Transaction, len(resp.Data))
	for i, t := range resp.Data {
		transactions[i] = mapTransaction(t)
	}

	return transactions, resp.NextCursor, resp.HasMore, nil
}

func (c *client) GetDeclinedTransactions(ctx context.Context, lastID string, pageSize int64) ([]*Transaction, string, bool, error) {
	ctx = context.WithValue(ctx, api.MetricOperationContextKey, "list_declined_transactions")

	params := &increase.TransactionListParams{
		Limit:  increase.F(int32(pageSize)),
		Status: increase.F(increase.TransactionListParamsStatusDeclined),
	}
	if lastID != "" {
		params.Cursor = increase.F(lastID)
	}

	resp, err := c.sdk.Transactions.List(ctx, params)
	if err != nil {
		return nil, "", false, err
	}

	transactions := make([]*Transaction, len(resp.Data))
	for i, t := range resp.Data {
		transactions[i] = mapTransaction(t)
	}

	return transactions, resp.NextCursor, resp.HasMore, nil
}
