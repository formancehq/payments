package client

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

type TokenBalance struct {
	MonitorID string    `json:"monitorId"`
	Asset     string    `json:"asset"`
	Amount    string    `json:"amount"`
	FetchedAt time.Time `json:"fetchedAt"`
}

// ParseAmount parses the string amount into a *big.Int.
func (b *TokenBalance) ParseAmount() (*big.Int, bool) {
	amount := new(big.Int)
	_, ok := amount.SetString(b.Amount, 10)
	return amount, ok
}

func (c *client) GetBalances(ctx context.Context) ([]*TokenBalance, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_balances")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildEndpoint("balances"), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var resp struct {
		Data []*TokenBalance `json:"data"`
	}
	var errRes chainbridgeError

	_, err = c.httpClient.Do(ctx, req, &resp, &errRes)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get balances: %s", errRes.ErrorMessage),
			err,
		)
	}

	return resp.Data, nil
}
