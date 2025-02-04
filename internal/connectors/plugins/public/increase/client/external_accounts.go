package client

import (
	"context"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/Increase/increase-go"
)

type ExternalAccount struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	AccountNumber string `json:"account_number"`
	RoutingNumber string `json:"routing_number"`
	Status        string `json:"status"`
	Type          string `json:"type"`
}

type CreateExternalAccountRequest struct {
	Name          string `json:"name"`
	AccountNumber string `json:"account_number"`
	RoutingNumber string `json:"routing_number"`
}

func mapExternalAccount(a *increase.ExternalAccount) *ExternalAccount {
	return &ExternalAccount{
		ID:            a.ID,
		Name:          a.Name,
		AccountNumber: a.AccountNumber,
		RoutingNumber: a.RoutingNumber,
		Status:        string(a.Status),
		Type:          string(a.Type),
	}
}

func (c *client) GetExternalAccounts(ctx context.Context, lastID string, pageSize int64) ([]*ExternalAccount, string, bool, error) {
	ctx = context.WithValue(ctx, api.MetricOperationContextKey, "list_external_accounts")

	params := &increase.ExternalAccountListParams{
		Limit: increase.F(int32(pageSize)),
	}
	if lastID != "" {
		params.Cursor = increase.F(lastID)
	}

	resp, err := c.sdk.ExternalAccounts.List(ctx, params)
	if err != nil {
		return nil, "", false, err
	}

	accounts := make([]*ExternalAccount, len(resp.Data))
	for i, a := range resp.Data {
		accounts[i] = mapExternalAccount(a)
	}

	return accounts, resp.NextCursor, resp.HasMore, nil
}

func (c *client) CreateExternalAccount(ctx context.Context, req *CreateExternalAccountRequest) (*ExternalAccount, error) {
	ctx = context.WithValue(ctx, api.MetricOperationContextKey, "create_external_account")

	params := &increase.ExternalAccountCreateParams{
		Name:          req.Name,
		AccountNumber: req.AccountNumber,
		RoutingNumber: req.RoutingNumber,
	}

	account, err := c.sdk.ExternalAccounts.New(ctx, params)
	if err != nil {
		return nil, err
	}

	return mapExternalAccount(account), nil
}
