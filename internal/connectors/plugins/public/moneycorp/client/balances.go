package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type balancesResponse struct {
	Balances []*Balance `json:"data"`
}

type Balance struct {
	ID         string     `json:"id"`
	Attributes Attributes `json:"attributes"`
}

type Attributes struct {
	CurrencyCode     string      `json:"currencyCode"`
	OverallBalance   json.Number `json:"overallBalance"`
	AvailableBalance json.Number `json:"availableBalance"`
	ClearedBalance   json.Number `json:"clearedBalance"`
	ReservedBalance  json.Number `json:"reservedBalance"`
	UnclearedBalance json.Number `json:"unclearedBalance"`
}

func (c *client) GetAccountBalances(ctx context.Context, accountID string) ([]*Balance, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_account_balances")

	endpoint := fmt.Sprintf("%s/accounts/%s/balances", c.endpoint, accountID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	balances := balancesResponse{Balances: make([]*Balance, 0)}
	var errRes moneycorpError

	_, err = c.httpClient.Do(ctx, req, &balances, &errRes)
	if err != nil {
		if errRes.StatusCode == http.StatusNotFound {
			// No balances found
			return []*Balance{}, nil
		}
		return nil, fmt.Errorf("failed to get account balances: %w %w", err, errRes.Error())
	}

	return balances.Balances, nil
}
