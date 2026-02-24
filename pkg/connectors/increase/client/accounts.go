package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/formancehq/payments/pkg/connector/metrics"
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

func (c *client) GetAccounts(ctx context.Context, pageSize int, cursor string, createdAtAfter time.Time) ([]*Account, string, error) {
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
	if !createdAtAfter.IsZero() && cursor == "" {
		q.Add("created_at.after", createdAtAfter.Format(time.RFC3339))
	}
	req.URL.RawQuery = q.Encode()

	var res ResponseWrapper[[]*Account]
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get accounts: %w %w", err, errRes.Error())
	}
	return res.Data, res.NextCursor, nil
}
