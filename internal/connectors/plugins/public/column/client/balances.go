package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Balance struct {
	AvailableAmount json.Number `json:"available_amount"`
	HoldingAmount   json.Number `json:"holding_amount"`
	LockedAmount    json.Number `json:"locked_amount"`
	PendingAmount   json.Number `json:"pending_amount"`
}

type BalanceResponseWrapper[t any] struct {
	Balances t `json:"balances"`
}

func (c *client) GetAccountBalances(ctx context.Context, bankAccountID string) (*Balance, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_account_balances")

	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("bank-accounts/%s", bankAccountID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create account balance request: %w", err)
	}

	var res BalanceResponseWrapper[*Balance]
	var errRes columnError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w %w", err, errRes.Error())
	}

	return res.Balances, nil
}
