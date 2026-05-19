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
	GetOpenOrders(ctx context.Context) ([]OpenOrder, error)
	GetOpenOrdersForMarket(ctx context.Context, currencyPair string) ([]OpenOrder, error)
	GetOrderStatus(ctx context.Context, orderID string) (OrderStatus, error)
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

// signedPOST is the shared shape for every authenticated v2 endpoint:
// optional form body, HMAC headers, JSON-or-error envelope response.
// It maps Bitstamp's API5506 derivatives-unsupported error to a typed
// DerivativesUnsupportedError so callers can spot-only-skip it.
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
		if errResp.Code == ErrCodeDerivativesUnsupported {
			return &DerivativesUnsupportedError{Endpoint: path, Message: errResp.Message()}
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

// GetOpenOrders returns the current snapshot of all open orders across
// every currency pair. The Bitstamp server caches this for ~10s, so
// back-to-back polls do not see fresher data.
func (c *client) GetOpenOrders(ctx context.Context) ([]OpenOrder, error) {
	var out []OpenOrder
	if err := c.signedPOST(ctx, "/api/v2/open_orders/all/", nil, &out); err != nil {
		return nil, fmt.Errorf("get open orders: %w", err)
	}
	return out, nil
}

// GetOpenOrdersForMarket is the per-pair variant; included for future
// per-market polling (the orders task uses GetOpenOrders today).
// Currency pair is normalised to lowercase per Bitstamp convention.
func (c *client) GetOpenOrdersForMarket(ctx context.Context, currencyPair string) ([]OpenOrder, error) {
	pair := strings.ToLower(strings.TrimSpace(currencyPair))
	if pair == "" {
		return nil, fmt.Errorf("get open orders: currency pair is required")
	}
	var out []OpenOrder
	if err := c.signedPOST(ctx, "/api/v2/open_orders/"+pair+"/", nil, &out); err != nil {
		return nil, fmt.Errorf("get open orders for %s: %w", pair, err)
	}
	return out, nil
}

// GetOrderStatus fetches the current status + fills for a single
// order. The order ID is sent in the form body (not the URL path) per
// Bitstamp's documented endpoint. Bitstamp retains order_status data
// for ~30 days; older IDs return an error, which the orders task
// guards against via the 25-day TrackedOrders eviction policy.
func (c *client) GetOrderStatus(ctx context.Context, orderID string) (OrderStatus, error) {
	if orderID == "" {
		return OrderStatus{}, fmt.Errorf("get order status: order id is required")
	}
	form := url.Values{}
	form.Set("id", orderID)
	form.Set("omit_transactions", "false")
	var out OrderStatus
	if err := c.signedPOST(ctx, "/api/v2/order_status/", form, &out); err != nil {
		return OrderStatus{}, fmt.Errorf("get order status %s: %w", orderID, err)
	}
	return out, nil
}
