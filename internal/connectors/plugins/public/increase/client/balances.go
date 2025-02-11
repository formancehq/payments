package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Balance struct {
	AccountID        string      `json:"account_id"`
	CurrentBalance   json.Number `json:"current_balance"`
	AvailableBalance json.Number `json:"available_balance"`
	Type             string      `json:"type"`
}

func (c *client) GetAccountBalance(ctx context.Context, accountID string) (*Balance, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_account_balances")

	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("accounts/%s/balance", accountID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create account balance request: %w", err)
	}

	var res *Balance
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get account balance: %w %w", err, errRes.Error())
	}
	return res, nil
}
