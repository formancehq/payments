package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

type BalanceAmount struct {
	Value    json.Number `json:"value"`
	Currency string      `json:"currency"`
}

type Balance struct {
	ID             uint64        `json:"id"`
	Currency       string        `json:"currency"`
	Type           string        `json:"type"`
	Name           string        `json:"name"`
	Amount         BalanceAmount `json:"amount"`
	ReservedAmount struct {
		Value    json.Number `json:"value"`
		Currency string      `json:"currency"`
	} `json:"reservedAmount"`
	CashAmount struct {
		Value    json.Number `json:"value"`
		Currency string      `json:"currency"`
	} `json:"cashAmount"`
	TotalWorth struct {
		Value    json.Number `json:"value"`
		Currency string      `json:"currency"`
	} `json:"totalWorth"`
	CreationTime     time.Time `json:"creationTime"`
	ModificationTime time.Time `json:"modificationTime"`
	Visible          bool      `json:"visible"`
}

func (c *client) GetBalances(ctx context.Context, profileID uint64) ([]Balance, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_balances")

	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet, c.endpoint(fmt.Sprintf("v4/profiles/%d/balances?types=STANDARD", profileID)), http.NoBody)
	if err != nil {
		return nil, err
	}

	var balances []Balance
	var errRes wiseErrors
	statusCode, err := c.httpClient.Do(ctx, req, &balances, &errRes)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get balances: %v", errRes.Error(statusCode)),
			err,
		)
	}
	return balances, nil
}

func (c *client) GetBalance(ctx context.Context, profileID uint64, balanceID uint64) (*Balance, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_balance")

	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet, c.endpoint(fmt.Sprintf("v4/profiles/%d/balances/%d", profileID, balanceID)), http.NoBody)
	if err != nil {
		return nil, err
	}

	var balance Balance
	var errRes wiseErrors
	statusCode, err := c.httpClient.Do(ctx, req, &balance, &errRes)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get balance: %v", errRes.Error(statusCode)),
			err,
		)
	}
	return &balance, nil
}
