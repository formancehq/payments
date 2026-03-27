package client

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/google/uuid"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetAccountBalances(ctx context.Context) ([]AccountBalance, error)
	GetUserTransactions(ctx context.Context, offset, limit int) ([]UserTransaction, error)
	GetCurrencies(ctx context.Context) ([]Currency, error)
}

const defaultBaseURL = "https://www.bitstamp.net"

type client struct {
	httpClient httpwrapper.Client
	baseURL    string
	apiKey     string
	apiSecret  string
}

func New(connectorName, apiKey, apiSecret string) Client {
	return NewWithBaseURL(connectorName, apiKey, apiSecret, defaultBaseURL)
}

func NewWithBaseURL(connectorName, apiKey, apiSecret, baseURL string) Client {
	c := &client{
		baseURL:   baseURL,
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}

	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{
			Transport: http.DefaultTransport,
		}),
	}
	c.httpClient = httpwrapper.NewClient(config)

	return c
}

func (c *client) signRequest(req *http.Request, body string) {
	nonce := uuid.New().String()
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	host := req.URL.Host
	path := req.URL.Path
	query := req.URL.RawQuery

	contentType := ""
	if body != "" {
		contentType = "application/x-www-form-urlencoded"
	}

	// Bitstamp v2 HMAC string-to-sign: raw concatenation, no separators
	message := "BITSTAMP " + c.apiKey +
		req.Method +
		host +
		path +
		query +
		contentType +
		nonce +
		timestamp +
		"v2" +
		body

	h := hmac.New(sha256.New, []byte(c.apiSecret))
	h.Write([]byte(message))
	signature := hex.EncodeToString(h.Sum(nil))

	req.Header.Set("X-Auth", "BITSTAMP "+c.apiKey)
	req.Header.Set("X-Auth-Signature", signature)
	req.Header.Set("X-Auth-Nonce", nonce)
	req.Header.Set("X-Auth-Timestamp", timestamp)
	req.Header.Set("X-Auth-Version", "v2")
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
}

func (c *client) GetAccountBalances(ctx context.Context) ([]AccountBalance, error) {
	endpoint := fmt.Sprintf("%s/api/v2/account_balances/", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.signRequest(req, "")

	var response []AccountBalance
	var errorResponse ErrorResponse
	statusCode, err := c.httpClient.Do(ctx, req, &response, &errorResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to get account balances (status %d, message: %s): %w", statusCode, errorResponse.Message(), err)
	}

	return response, nil
}

func (c *client) GetUserTransactions(ctx context.Context, offset, limit int) ([]UserTransaction, error) {
	endpoint := fmt.Sprintf("%s/api/v2/user_transactions/", c.baseURL)

	params := url.Values{}
	params.Set("offset", strconv.Itoa(offset))
	params.Set("limit", strconv.Itoa(limit))
	params.Set("sort", "asc")
	body := params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.signRequest(req, body)

	var response []UserTransaction
	var errorResponse ErrorResponse
	statusCode, err := c.httpClient.Do(ctx, req, &response, &errorResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to get user transactions (status %d, message: %s): %w", statusCode, errorResponse.Message(), err)
	}

	return response, nil
}

func (c *client) GetCurrencies(ctx context.Context) ([]Currency, error) {
	endpoint := fmt.Sprintf("%s/api/v2/currencies/", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Public endpoint — no auth needed

	var response []Currency
	var errorResponse ErrorResponse
	statusCode, err := c.httpClient.Do(ctx, req, &response, &errorResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to get currencies (status %d, message: %s): %w", statusCode, errorResponse.Message(), err)
	}

	return response, nil
}

// AccountBalance represents a Bitstamp account balance for a single currency.
type AccountBalance struct {
	Currency  string `json:"currency"`
	Total     string `json:"total"`
	Available string `json:"available"`
	Reserved  string `json:"reserved"`
}

// UserTransaction represents a Bitstamp user transaction with dynamic currency keys.
type UserTransaction struct {
	ID              int64             `json:"id"`
	Datetime        string            `json:"datetime"`
	Type            string            `json:"type"`
	Fee             string            `json:"fee"`
	OrderID         int64             `json:"order_id"`
	CurrencyAmounts map[string]string `json:"-"`
}

func (ut *UserTransaction) UnmarshalJSON(data []byte) error {
	// First unmarshal known fields via an alias to avoid recursion.
	type Alias UserTransaction
	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*ut = UserTransaction(alias)

	// Then extract dynamic currency keys from the raw map.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	knownKeys := map[string]struct{}{
		"id": {}, "datetime": {}, "type": {}, "fee": {},
		"order_id": {}, "self_trade": {}, "self_trade_order_id": {},
		"status": {}, "reason": {}, "market": {},
	}

	ut.CurrencyAmounts = make(map[string]string)
	for key, val := range raw {
		if _, known := knownKeys[key]; known {
			continue
		}

		// Try to parse as a string (most currency amounts are strings).
		var strVal string
		if err := json.Unmarshal(val, &strVal); err == nil {
			// Verify it looks like a decimal number.
			if _, ok := new(big.Float).SetString(strVal); ok {
				ut.CurrencyAmounts[key] = strVal
			}
			continue
		}

		// Try to parse as a number (some fields like exchange rates are numbers).
		var numVal float64
		if err := json.Unmarshal(val, &numVal); err == nil {
			ut.CurrencyAmounts[key] = strconv.FormatFloat(numVal, 'f', -1, 64)
		}
	}

	return nil
}

// Currency represents a Bitstamp currency with its decimal precision.
type Currency struct {
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Decimals int    `json:"decimals"`
	Type     string `json:"type"`
}

// ErrorResponse represents a Bitstamp API error.
// Bitstamp uses two formats: old (status/reason/code) and new (code/message).
type ErrorResponse struct {
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Code    string `json:"code"`
	Msg     string `json:"message"`
}

func (e ErrorResponse) Message() string {
	if e.Msg != "" {
		return e.Msg
	}
	if e.Reason != "" {
		return e.Reason
	}
	return e.Code
}
