package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type BankAccountRequest struct {
	AccountHolder string `json:"account_holder"`
	AccountNumber string `json:"account_number"`
	Description   string `json:"description"`
	RoutingNumber string `json:"routing_number"`
}

type BankAccountResponse struct {
	ID            string `json:"id"`
	CreatedAt     string `json:"created_at"`
	Description   string `json:"description"`
	Status        string `json:"status"`
	RoutingNumber string `json:"routing_number"`
	AccountNumber string `json:"account_number"`
	Type          string `json:"type"`
	AccountHolder string `json:"account_holder"`
}

func (c *client) CreateBankAccount(ctx context.Context, pr *BankAccountRequest) (*BankAccountResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_external_account")

	body, err := json.Marshal(pr)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "external_accounts", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create external account request: %w", err)
	}

	var res BankAccountResponse
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to create external account: %w %w", err, errRes.Error())
	}

	return &res, nil
}
