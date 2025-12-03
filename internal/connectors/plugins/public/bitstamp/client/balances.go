package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

// Balance represents a currency balance in a Bitstamp account.
// All amounts are returned as strings to preserve decimal precision.
type Balance struct {
	Currency  string `json:"currency"`  // e.g., "usd", "btc", "eur"
	Total     string `json:"total"`     // Total balance
	Available string `json:"available"` // Available for trading
	Reserved  string `json:"reserved"`  // Locked in open orders
}

// bitstampErrors captures API error responses
type bitstampErrors struct {
	Error string `json:"error"`
}

// GetAccountBalances fetches all currency balances for a specific Bitstamp account
// using the account_balances endpoint. The API returns one balance entry per currency
// that has ever had a non-zero balance in the account.
func (c *client) GetAccountBalances(ctx context.Context, account *Account) ([]*Balance, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_account_balances")

	httpClient := c.httpClientForAccount(account)
	endpoint := "https://www.bitstamp.net/api/v2/account_balances/"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(""))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	var balances []*Balance
	var errRes bitstampErrors
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try to unmarshal error response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("bitstamp: status %d: failed to read response body: %w", resp.StatusCode, err)
		}

		if err := json.Unmarshal(body, &errRes); err == nil && errRes.Error != "" {
			return nil, fmt.Errorf("bitstamp: status %d: %s", resp.StatusCode, errRes.Error)
		}

		return nil, fmt.Errorf("bitstamp: status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &balances); err != nil {
		return nil, fmt.Errorf("bitstamp: decode error: %w", err)
	}

	return balances, nil
}
