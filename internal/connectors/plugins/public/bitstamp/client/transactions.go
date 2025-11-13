package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Transaction represents a Bitstamp user transaction (trade, deposit, withdrawal, etc).
// Currency fields are dynamic - Bitstamp includes columns for each currency traded
// (e.g., EUR, USD, BTC, USDC). Non-zero values indicate which currencies were affected.
type Transaction struct {
	ID       NumString `json:"id"`       // e.g., "458254264" or 458254264 → captured as "458254264"
	Datetime string    `json:"datetime"` // "2025-09-25 14:42:59.894846"
	Type     NumString `json:"type"`     // e.g., "36"
	Fee      NumString `json:"fee"`      // "0.00000"
	EUR      NumString `json:"eur"`      // "-5.00"
	USDC     NumString `json:"usdc"`     // "5.81077"
	USD      NumString `json:"usd"`      // "0.0"
	BTC      NumString `json:"btc"`      // "0.0"
	DOGE     NumString `json:"doge"`     // "0.0"

	// Rates captures all exchange rate fields dynamically (btc_eur, usdc_eur, etc.)
	Rates map[string]NumString `json:"-"`
}

// UnmarshalJSON custom unmarshaler to capture exchange rate fields dynamically.
func (t *Transaction) UnmarshalJSON(data []byte) error {
	type Alias Transaction
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(t),
	}

	// First unmarshal into a map to get all fields
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Then unmarshal known fields into the struct
	if err := json.Unmarshal(data, &aux.Alias); err != nil {
		return err
	}

	// Capture exchange rate fields (those with underscores)
	t.Rates = make(map[string]NumString)
	for key, val := range raw {
		if strings.Contains(key, "_") {
			var ns NumString
			if err := json.Unmarshal(val, &ns); err == nil {
				t.Rates[key] = ns
			}
		}
	}

	return nil
}

// GetExchangeRate returns the non-zero exchange rate for this transaction, if any.
func (t *Transaction) GetExchangeRate() (pair string, rate NumString) {
	for pair, rate := range t.Rates {
		if rate != "0" && rate != "0.0" && rate != "0.00" && rate != "" {
			return pair, rate
		}
	}
	return "", ""
}

// GetTransactions fetches transactions using the "main" account credentials.
// This is a convenience method for single-account setups.
func (c *client) GetTransactions(ctx context.Context, p TransactionsParams) ([]Transaction, error) {
	mainAccount, err := c.getMainAccount()
	if err != nil {
		return nil, err
	}
	return c.GetTransactionsForAccount(ctx, mainAccount, p)
}

// GetTransactionsForAccount fetches transactions for a specific Bitstamp account.
// The method supports pagination (offset/limit), time windows (since/until timestamps),
// and sorting. Maximum 1000 transactions per request per Bitstamp API limits.
func (c *client) GetTransactionsForAccount(ctx context.Context, account *Account, p TransactionsParams) ([]Transaction, error) {
	form := url.Values{}

	// ----- offset -----
	offset := p.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > 200000 {
		offset = 200000
	}
	form.Set("offset", strconv.Itoa(offset))

	// ----- limit (+ since_id rule) -----
	limit := p.Limit
	if p.SinceID != "" {
		limit = 1000 // per docs
	}
	if limit < 10 { // Use a reasonable minimum (handles 0, negative, and small values)
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	form.Set("limit", strconv.Itoa(limit))

	// ----- sort -----
	sort := strings.ToLower(strings.TrimSpace(p.Sort))
	if sort != "asc" && sort != "desc" {
		sort = "desc"
	}
	form.Set("sort", sort)

	// ----- since/until timestamps (unix seconds) -----
	if p.SinceTimestamp > 0 {
		form.Set("since_timestamp", strconv.FormatInt(p.SinceTimestamp, 10))
	}
	if p.UntilTimestamp > 0 {
		form.Set("until_timestamp", strconv.FormatInt(p.UntilTimestamp, 10))
	}

	// ----- since_id (optional) -----
	if p.SinceID != "" {
		form.Set("since_id", p.SinceID)
	}

	endpoint := "https://www.bitstamp.net/api/v2/user_transactions/"
	bodyStr := form.Encode()

	// Log the request details
	fmt.Printf("[BITSTAMP] GetTransactionsForAccount - Account: %s (ID: %s)\n", account.Name, account.ID)
	fmt.Printf("[BITSTAMP] Endpoint: %s\n", endpoint)
	fmt.Printf("[BITSTAMP] Request body: %s\n", bodyStr)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(bodyStr))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Authenticate with the specific account's credentials
	httpClient := c.httpClientForAccount(account)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[BITSTAMP] Error response: status %d: %s\n", resp.StatusCode, string(data))
		return nil, fmt.Errorf("bitstamp: status %d: %s", resp.StatusCode, string(data))
	}

	var txs []Transaction
	if err := json.Unmarshal(data, &txs); err != nil {
		fmt.Printf("[BITSTAMP] Decode error: %v; body=%s\n", err, string(data))
		return nil, fmt.Errorf("bitstamp: decode error: %w; body=%s", err, string(data))
	}

	fmt.Printf("[BITSTAMP] Successfully fetched %d transactions\n", len(txs))

	return txs, nil
}

// (Optional) Back-compat helper if you still call with just an offset string/int:
func (c *client) UserTransactionsOffset(ctx context.Context, offset int) ([]Transaction, error) {
	return c.GetTransactions(ctx, TransactionsParams{Offset: offset})
}

// Example helper to build a 7-day window:
func SevenDayWindow() (since, until int64) {
	now := time.Now().Unix()
	day := int64(24 * 60 * 60)
	return now - 7*day, now
}

// NumString handles JSON values that can be either strings or numbers.
// Bitstamp inconsistently returns some numeric fields as strings vs numbers.
type NumString string

// UnmarshalJSON normalizes both quoted and unquoted numbers to strings.
func (n *NumString) UnmarshalJSON(b []byte) error {
	// If it's quoted, strip quotes; else keep numeric literal as-is.
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*n = NumString(s)
		return nil
	}
	*n = NumString(string(b)) // number → keep literal "0.0", "0", "0.86047000", etc.
	return nil
}
