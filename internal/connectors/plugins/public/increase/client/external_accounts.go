package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/pkg/metrics"
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

func (c *client) GetExternalAccounts(ctx context.Context, lastID string, pageSize int64) ([]*ExternalAccount, string, bool, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_external_accounts")

	endpoint := fmt.Sprintf("/external_accounts?limit=%d", pageSize)
	if lastID != "" {
		endpoint += "&cursor=" + lastID
	}

	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", false, err
	}

	var response struct {
		Data     []*ExternalAccount `json:"data"`
		NextPage string             `json:"next_page"`
	}
	if err := c.do(req, &response); err != nil {
		return nil, "", false, err
	}

	return response.Data, response.NextPage, response.NextPage != "", nil
}

func (c *client) CreateExternalAccount(ctx context.Context, req *CreateExternalAccountRequest) (*ExternalAccount, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_external_account")

	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, "/external_accounts", body)
	if err != nil {
		return nil, err
	}

	var account ExternalAccount
	if err := c.do(httpReq, &account); err != nil {
		return nil, err
	}

	return &account, nil
}
