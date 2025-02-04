package client

import (
	"context"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/Increase/increase-go"
)

type Account struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Type      string    `json:"type"`
	Currency  string    `json:"currency"`
	Bank      string    `json:"bank"`
	CreatedAt time.Time `json:"created_at"`
}

func mapAccount(a *increase.Account) *Account {
	return &Account{
		ID:        a.ID,
		Name:      a.Name,
		Status:    string(a.Status),
		Type:      string(a.Type),
		Currency:  string(a.Currency),
		Bank:      string(a.Bank),
		CreatedAt: a.CreatedAt,
	}
}

func (c *client) GetAccounts(ctx context.Context, lastID string, pageSize int64) ([]*Account, string, bool, error) {
	ctx = context.WithValue(ctx, api.MetricOperationContextKey, "list_accounts")

	params := &increase.AccountListParams{
		Limit: increase.F(int32(pageSize)),
	}
	if lastID != "" {
		params.Cursor = increase.F(lastID)
	}

	resp, err := c.sdk.Accounts.List(ctx, params)
	if err != nil {
		return nil, "", false, err
	}

	accounts := make([]*Account, len(resp.Data))
	for i, a := range resp.Data {
		accounts[i] = mapAccount(a)
	}

	return accounts, resp.NextCursor, resp.HasMore, nil
}
