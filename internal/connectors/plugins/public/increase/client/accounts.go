package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Account struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	EntityID  string `json:"entity_id"`
	Bank      string `json:"bank"`
	Status    string `json:"status"`
	Type      string `json:"type"`
	Currency  string `json:"currency"`
	CreatedAt string `json:"created_at"`
}

func (c *client) GetAccounts(ctx context.Context, pageSize int, cursor string) ([]*Account, string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_accounts")

	req, err := c.newRequest(ctx, http.MethodGet, "accounts", http.NoBody)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create account request: %w", err)
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(pageSize))
	if cursor != "" {
		q.Add("cursor", cursor)
	}
	req.URL.RawQuery = q.Encode()

	var res responseWrapper[[]*Account]
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get accounts: %w %w", err, errRes.Error())
	}
	return res.Data, res.NextCursor, nil
}

func (c *client) GetAccount(ctx context.Context, accountID string) (*Account, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_account")

	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("accounts/%s", accountID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create account request: %w", err)
	}

	var res Account
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w %w", err, errRes.Error())
	}
	return &res, nil
}
