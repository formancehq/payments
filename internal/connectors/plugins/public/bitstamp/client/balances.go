package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

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

	log.Printf("[BITSTAMP] GetAccountBalances called for account: %s (name: %s)", account.ID, account.Name)

	httpClient := c.httpClientForAccount(account)
	endpoint := "https://www.bitstamp.net/api/v2/account_balances/"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, http.NoBody)
	if err != nil {
		log.Printf("[BITSTAMP] Error creating request: %v", err)
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	log.Printf("[BITSTAMP] Sending POST request to %s", endpoint)
	log.Printf("[BITSTAMP] Request headers: %+v", req.Header)

	var balances []*Balance
	var errRes bitstampErrors
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("[BITSTAMP] Error making request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("[BITSTAMP] Response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		// Try to unmarshal error response
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[BITSTAMP] Error response body: %s", string(body))
		json.Unmarshal(body, &errRes)
		return nil, fmt.Errorf("bitstamp: status %d: %s", resp.StatusCode, errRes.Error)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[BITSTAMP] Error reading response body: %v", err)
		return nil, err
	}

	log.Printf("[BITSTAMP] Response body: %s", string(body))

	if err := json.Unmarshal(body, &balances); err != nil {
		log.Printf("[BITSTAMP] Error unmarshaling response: %v", err)
		return nil, fmt.Errorf("bitstamp: decode error: %w", err)
	}

	log.Printf("[BITSTAMP] Successfully fetched %d balances", len(balances))

	return balances, nil
}
