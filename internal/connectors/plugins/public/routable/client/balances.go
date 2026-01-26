package client

import (
	"context"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Balance struct {
	Currency        string `json:"currency"`
	AvailableAmount string `json:"available_amount"`
	PendingAmount   string `json:"pending_amount"`
	IsUsable        bool   `json:"is_usable"`
}

type balanceResponse struct {
	Object      string `json:"object"`
	ID          string `json:"id"`
	CreatedAt   string `json:"created_at"`
	Type        string `json:"type"`
	TypeDetails struct {
		Currency        string `json:"currency"`
		AvailableAmount string `json:"available_amount"`
		PendingAmount   string `json:"pending_amount"`
		IsUsable        bool   `json:"is_usable"`
	} `json:"type_details"`
}

func (c *client) GetAccountBalances(ctx context.Context) ([]*Balance, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "retrieve_balance")

	req, err := c.newRequest(ctx, http.MethodGet, "/v1/settings/balance", nil)
	if err != nil {
		return nil, err
	}

	var out balanceResponse
	if _, err := c.httpClient.Do(ctx, req, &out, &out); err != nil {
		return nil, err
	}

	currencyCode := out.TypeDetails.Currency
	if currencyCode == "" {
		currencyCode = "USD"
	}

	return []*Balance{{
		Currency:        currencyCode,
		AvailableAmount: out.TypeDetails.AvailableAmount,
		PendingAmount:   out.TypeDetails.PendingAmount,
		IsUsable:        out.TypeDetails.IsUsable,
	}}, nil
}
