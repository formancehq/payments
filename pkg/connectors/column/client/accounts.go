package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/formancehq/payments/pkg/connector/metrics"
)

type Balances struct {
	AvailableAmount json.Number `json:"available_amount"`
	HoldingAmount   json.Number `json:"holding_amount"`
	LockedAmount    json.Number `json:"locked_amount"`
	PendingAmount   json.Number `json:"pending_amount"`
}

type Account struct {
	Balances                  Balances
	Bic                       string
	ID                        string   `json:"id"`
	Type                      string   `json:"type"`
	CurrencyCode              string   `json:"currency_code"`
	DefaultAccountNumber      string   `json:"default_account_number"`
	DefaultAccountNumberID    string   `json:"default_account_number_id"`
	Description               string   `json:"description"`
	IsOverdraftable           bool     `json:"is_overdraftable"`
	OverdraftReserveAccountID string   `json:"overdraft_reserve_account_id"`
	RoutingNumber             string   `json:"routing_number"`
	Owners                    []string `json:"owners"`
	CreatedAt                 string   `json:"created_at"`
}

type AccountResponseWrapper[t any] struct {
	BankAccounts t    `json:"bank_accounts"`
	HasMore      bool `json:"has_more"`
}

func (c *client) GetAccounts(ctx context.Context, cursor string, pageSize int) ([]*Account, bool, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_accounts")

	req, err := c.newRequest(ctx, http.MethodGet, "bank-accounts", http.NoBody)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create account request: %w", err)
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(pageSize))
	if cursor != "" {
		q.Add("starting_after", cursor)
	}
	req.URL.RawQuery = q.Encode()

	var res AccountResponseWrapper[[]*Account]
	var errRes columnError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get accounts: %w %w", err, errRes.Error())
	}

	return res.BankAccounts, res.HasMore, nil
}
