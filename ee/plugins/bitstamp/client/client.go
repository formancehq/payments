package client

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetAccountBalances(ctx context.Context) ([]AccountBalance, error)
	GetUserTransactions(ctx context.Context, sinceID *int64, limit int) ([]UserTransaction, error)
	GetCurrencies(ctx context.Context) ([]Currency, error)
	GetAccountOrderData(ctx context.Context, market string, sinceID *string) ([]AccountOrderDataEvent, error)

	// Install-time enrichment endpoints — see MAPPINGS §12.2.
	GetMarkets(ctx context.Context) ([]Market, error)
	GetMyMarkets(ctx context.Context) ([]MyMarket, error)
	GetTradingFees(ctx context.Context) ([]TradingFee, error)
	GetWithdrawalFees(ctx context.Context) ([]WithdrawalFee, error)
}

const DefaultEndpoint = "https://www.bitstamp.net"

type client struct {
	httpClient httpwrapper.Client
	endpoint   string
	apiKey     string
	apiSecret  string
}

// New returns a Bitstamp REST v2 client. The HTTP transport is wrapped
// with otelhttp (per-request spans) and metrics (per-connector
// counters), in that order, so traces carry the connector name out of
// the box.
func New(connectorName, apiKey, apiSecret, endpoint string) Client {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	c := &client{
		endpoint:  strings.TrimRight(endpoint, "/"),
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}
	c.httpClient = httpwrapper.NewClient(&httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}),
	})
	return c
}

// signRequest applies Bitstamp v2 HMAC-SHA256 auth headers.
// The message-to-sign is a raw concatenation (no separators):
//
//	"BITSTAMP " + apiKey + method + host + path + query + contentType +
//	    nonce + timestamp + "v2" + body
//
// Bitstamp omits Content-Type for empty-body POSTs; signRequest mirrors
// that by reading req.Header.Get("Content-Type") (empty string when
// unset), so callers must NOT set Content-Type for empty bodies.
func (c *client) signRequest(req *http.Request, body string) {
	nonce := uuid.New().String()
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	message := "BITSTAMP " + c.apiKey +
		req.Method +
		req.URL.Host +
		req.URL.Path +
		req.URL.RawQuery +
		req.Header.Get("Content-Type") +
		nonce +
		timestamp +
		"v2" +
		body

	mac := hmac.New(sha256.New, []byte(c.apiSecret))
	mac.Write([]byte(message))

	req.Header.Set("X-Auth", "BITSTAMP "+c.apiKey)
	req.Header.Set("X-Auth-Signature", hex.EncodeToString(mac.Sum(nil)))
	req.Header.Set("X-Auth-Nonce", nonce)
	req.Header.Set("X-Auth-Timestamp", timestamp)
	req.Header.Set("X-Auth-Version", "v2")
}

// signedGET issues an authenticated GET with HMAC headers and no body.
func (c *client) signedGET(ctx context.Context, path string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+path, nil)
	if err != nil {
		return fmt.Errorf("create request %s: %w", path, err)
	}
	c.signRequest(req, "")
	var errResp ErrorResponse
	statusCode, err := c.httpClient.Do(ctx, req, dst, &errResp)
	if err != nil {
		if statusCode == 404 {
			return &NotFoundError{Endpoint: path, Message: errResp.Message()}
		}
		if errResp.Code == ErrCodeDerivativesUnsupported {
			return &DerivativesUnsupportedError{Endpoint: path, Message: errResp.Message()}
		}
		return fmt.Errorf("%s (status %d, message: %s): %w", path, statusCode, errResp.Message(), err)
	}
	return nil
}

// signedPOST is the shared shape for authenticated v2 POST endpoints:
// optional form body, HMAC headers, JSON-or-error envelope response.
func (c *client) signedPOST(ctx context.Context, path string, form url.Values, dst any) error {
	endpoint := c.endpoint + path
	var (
		body   string
		reader io.Reader
	)
	if form != nil {
		body = form.Encode()
		reader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, reader)
	if err != nil {
		return fmt.Errorf("create request %s: %w", path, err)
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	c.signRequest(req, body)

	var errResp ErrorResponse
	statusCode, err := c.httpClient.Do(ctx, req, dst, &errResp)
	if err != nil {
		if statusCode == 404 {
			return &NotFoundError{Endpoint: path, Message: errResp.Message()}
		}
		return fmt.Errorf("%s (status %d, message: %s): %w", path, statusCode, errResp.Message(), err)
	}
	return nil
}

func (c *client) GetAccountBalances(ctx context.Context) ([]AccountBalance, error) {
	var out []AccountBalance
	if err := c.signedPOST(ctx, "/api/v2/account_balances/", nil, &out); err != nil {
		return nil, fmt.Errorf("get account balances: %w", err)
	}
	return out, nil
}

func (c *client) GetUserTransactions(ctx context.Context, sinceID *int64, limit int) ([]UserTransaction, error) {
	form := url.Values{}
	form.Set("sort", "asc")
	if limit > 0 {
		form.Set("limit", strconv.Itoa(limit))
	}
	if sinceID != nil && *sinceID > 0 {
		form.Set("since_id", strconv.FormatInt(*sinceID, 10))
	}
	var out []UserTransaction
	if err := c.signedPOST(ctx, "/api/v2/user_transactions/", form, &out); err != nil {
		return nil, fmt.Errorf("get user transactions: %w", err)
	}
	return out, nil
}

// GetCurrencies hits the public /currencies/ endpoint — no auth, no
// signing. Kept on the same client so the metrics + otel transport
// instruments it uniformly.
func (c *client) GetCurrencies(ctx context.Context) ([]Currency, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/api/v2/currencies/", nil)
	if err != nil {
		return nil, fmt.Errorf("get currencies: create request: %w", err)
	}
	var out []Currency
	var errResp ErrorResponse
	statusCode, err := c.httpClient.Do(ctx, req, &out, &errResp)
	if err != nil {
		return nil, fmt.Errorf("get currencies (status %d, message: %s): %w", statusCode, errResp.Message(), err)
	}
	return out, nil
}

// GetAccountOrderData returns order lifecycle events for one market.
// order_source is always "orderbook". sinceID, when non-nil and positive,
// restricts the response to events whose order ID is greater than sinceID.
func (c *client) GetAccountOrderData(ctx context.Context, market string, sinceID *string) ([]AccountOrderDataEvent, error) {
	form := url.Values{}
	form.Set("order_source", "orderbook")
	form.Set("market", strings.TrimSpace(market))
	if sinceID != nil && *sinceID != "" {
		form.Set("since_id", *sinceID)
	}
	var out []AccountOrderDataEvent
	if err := c.signedPOST(ctx, "/api/v2/account_order_data/", form, &out); err != nil {
		return nil, fmt.Errorf("get account order data for %s: %w", market, err)
	}
	return out, nil
}

// GetMarkets returns every Bitstamp market (pair) with base/counter
// decimals, minimum order value, and market_type (SPOT vs derivatives
// variants — the spot-only enrichment filters non-SPOT rows out).
// Public GET; no signing required.
func (c *client) GetMarkets(ctx context.Context) ([]Market, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/api/v2/markets/", nil)
	if err != nil {
		return nil, fmt.Errorf("get markets: create request: %w", err)
	}
	var out []Market
	var errResp ErrorResponse
	statusCode, err := c.httpClient.Do(ctx, req, &out, &errResp)
	if err != nil {
		return nil, fmt.Errorf("get markets (status %d, message: %s): %w", statusCode, errResp.Message(), err)
	}
	return out, nil
}

// GetMyMarkets returns the trading-pair allow-list for the
// authenticated API key. Authenticated GET endpoint.
func (c *client) GetMyMarkets(ctx context.Context) ([]MyMarket, error) {
	var out []MyMarket
	if err := c.signedGET(ctx, "/api/v2/my_markets/", &out); err != nil {
		return nil, fmt.Errorf("get my markets: %w", err)
	}
	return out, nil
}

// GetTradingFees returns the maker/taker fee schedule for every pair
// the authenticated key can trade. The fee rate is a string-decimal
// percentage (e.g. "0.300" = 0.3%).
func (c *client) GetTradingFees(ctx context.Context) ([]TradingFee, error) {
	var out []TradingFee
	if err := c.signedPOST(ctx, "/api/v2/fees/trading/", nil, &out); err != nil {
		return nil, fmt.Errorf("get trading fees: %w", err)
	}
	return out, nil
}

// GetWithdrawalFees returns per-currency × per-network withdrawal
// fees (e.g. BTC has both `bitcoin: 0.00008` and `xrpl: 0`). Multiple
// rows per currency are expected on assets that span multiple
// blockchains (USDC, ETH, …).
func (c *client) GetWithdrawalFees(ctx context.Context) ([]WithdrawalFee, error) {
	var out []WithdrawalFee
	if err := c.signedPOST(ctx, "/api/v2/fees/withdrawal/", nil, &out); err != nil {
		return nil, fmt.Errorf("get withdrawal fees: %w", err)
	}
	return out, nil
}
