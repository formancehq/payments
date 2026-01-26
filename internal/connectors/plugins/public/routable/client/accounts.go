package client

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Account struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	Type      string `json:"type"`
}

type balanceAccountResponse struct {
	Object string `json:"object"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	// created_at is required but we keep string to let upper layer parse time layout
	CreatedAt string `json:"created_at"`
	Type      string `json:"type"`
}

func (c *client) GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "retrieve_balance_account")

	// We only have a single balance account to return
	if page > 0 {
		return []*Account{}, nil
	}

	req, err := c.newRequest(ctx, http.MethodGet, "/v1/settings/balance", nil)
	if err != nil {
		return nil, err
	}

	var out balanceAccountResponse
	if _, err := c.httpClient.Do(ctx, req, &out, &out); err != nil {
		return nil, err
	}

	raw, _ := json.Marshal(out)
	acc := &Account{
		ID:        out.ID,
		Name:      out.Name,
		CreatedAt: out.CreatedAt,
		Type:      out.Type,
	}
	_ = raw

	return []*Account{acc}, nil
}
