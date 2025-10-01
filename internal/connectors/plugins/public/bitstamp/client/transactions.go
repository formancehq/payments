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

type Transaction struct {
	ID       NumString `json:"id"`       // e.g., "458254264" or 458254264 → captured as "458254264"
	Datetime string    `json:"datetime"` // "2025-09-25 14:42:59.894846"
	Type     NumString `json:"type"`     // e.g., "36"
	Fee      NumString `json:"fee"`      // "0.00000"
	EUR      NumString `json:"eur"`      // "-5.00"
	USDC     NumString `json:"usdc"`     // "5.81077"
	USDCEUR  NumString `json:"usdc_eur"` // may be number or string → captured as "0.86047000"
	USD      NumString `json:"usd"`      // "0.0"
	BTC      NumString `json:"btc"`      // "0.0"
}

func (c *client) GetTransactions(ctx context.Context, p TransactionsParams) ([]Transaction, error) {
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
	if limit <= 0 {
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

	// Build the HTTP request. IMPORTANT: set Content-Type so the signing transport
	// includes it in the signature exactly as sent.
	endpoint := "https://www.bitstamp.net/api/v2/user_transactions/"
	bodyStr := form.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(bodyStr))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Optional: set ContentLength to avoid chunked encoding (not required)
	// req.ContentLength = int64(len(bodyStr))

	// Use your configured http.Client that has the signing RoundTripper.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bitstamp: status %d: %s", resp.StatusCode, string(data))
	}

	var txs []Transaction
	if err := json.Unmarshal(data, &txs); err != nil {
		return nil, fmt.Errorf("bitstamp: decode error: %w; body=%s", err, string(data))
	}

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

type NumString string

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
