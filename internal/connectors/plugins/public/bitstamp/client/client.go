package client

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/google/uuid"
)

const (
	defaultTimeout = 30 * time.Second
	baseURL        = "https://www.bitstamp.net/api/v2"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	// Balance operations
	GetBalances(ctx context.Context) ([]AccountBalance, error)

	// Order operations
	GetOpenOrders(ctx context.Context, market string) ([]Order, error)
	GetOrderStatus(ctx context.Context, orderID string) (*OrderStatus, error)
	CreateLimitBuyOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error)
	CreateLimitSellOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error)
	CreateMarketBuyOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error)
	CreateMarketSellOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error)
	CancelOrder(ctx context.Context, orderID string) (*CancelOrderResponse, error)

	// Asset operations
	GetTradingPairs(ctx context.Context) ([]TradingPair, error)

	// Market data operations (public, no auth required)
	GetOrderBook(ctx context.Context, market string) (*OrderBookResponse, error)
	GetTicker(ctx context.Context, market string) (*TickerResponse, error)
	GetOHLC(ctx context.Context, market string, step int, limit int) (*OHLCResponse, error)
}

type client struct {
	httpClient *http.Client
	apiKey     string
	apiSecret  string
}

func New(
	connectorName string,
	apiKey string,
	apiSecret string,
) Client {
	httpClient := metrics.NewHTTPClient(connectorName, defaultTimeout)

	return &client{
		httpClient: httpClient,
		apiKey:     apiKey,
		apiSecret:  apiSecret,
	}
}

// generateNonce creates a unique nonce for each request
func generateNonce() string {
	return uuid.New().String()
}

// generateTimestamp returns the current UNIX timestamp in milliseconds
func generateTimestamp() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 10)
}

// sign creates the API signature for Bitstamp authentication
// Bitstamp uses HMAC-SHA256 for API v2
func (c *client) sign(method, path, contentType string, nonce, timestamp string, body string) string {
	// Create the message to sign
	// Format: BITSTAMP {api_key} {http_method} {host} {path} {query_string} {content_type} {nonce} {timestamp} v2 {body}
	host := "www.bitstamp.net"
	queryString := ""

	message := fmt.Sprintf("BITSTAMP %s%s%s%s%s%s%s%sv2%s",
		c.apiKey,
		strings.ToUpper(method),
		host,
		path,
		queryString,
		contentType,
		nonce,
		timestamp,
		body,
	)

	// Create HMAC-SHA256 signature
	h := hmac.New(sha256.New, []byte(c.apiSecret))
	h.Write([]byte(message))
	signature := hex.EncodeToString(h.Sum(nil))

	return strings.ToUpper(signature)
}

// doPrivateRequest performs an authenticated request
func (c *client) doPrivateRequest(ctx context.Context, method, path string, params url.Values) ([]byte, error) {
	nonce := generateNonce()
	timestamp := generateTimestamp()

	body := ""
	if params != nil {
		body = params.Encode()
	}

	// Only set Content-Type when there's a body - Bitstamp rejects requests with
	// Content-Type header when there's no body (API0020 error)
	contentType := ""
	if body != "" {
		contentType = "application/x-www-form-urlencoded"
	}

	// The signature must use the full path including /api/v2 prefix
	fullPath := "/api/v2" + path
	signature := c.sign(method, fullPath, contentType, nonce, timestamp, body)

	reqURL := baseURL + path
	var req *http.Request
	var err error

	if method == http.MethodPost && body != "" {
		req, err = http.NewRequestWithContext(ctx, method, reqURL, strings.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, reqURL, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Content-Type header only when there's a body
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("X-Auth", fmt.Sprintf("BITSTAMP %s", c.apiKey))
	req.Header.Set("X-Auth-Signature", signature)
	req.Header.Set("X-Auth-Nonce", nonce)
	req.Header.Set("X-Auth-Timestamp", timestamp)
	req.Header.Set("X-Auth-Version", "v2")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	// Check for API errors
	var errResp struct {
		Status string `json:"status"`
		Reason interface{} `json:"reason"`
		Code   string `json:"code"`
	}
	if json.Unmarshal(respBody, &errResp) == nil && errResp.Status == "error" {
		return nil, fmt.Errorf("bitstamp API error: %v (code: %s)", errResp.Reason, errResp.Code)
	}

	return respBody, nil
}

// doPublicRequest performs an unauthenticated request to a public endpoint
func (c *client) doPublicRequest(ctx context.Context, path string) ([]byte, error) {
	reqURL := baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (c *client) GetBalances(ctx context.Context) ([]AccountBalance, error) {
	body, err := c.doPrivateRequest(ctx, http.MethodPost, "/account_balances/", nil)
	if err != nil {
		return nil, err
	}

	var balances []AccountBalance
	if err := json.Unmarshal(body, &balances); err != nil {
		return nil, fmt.Errorf("failed to unmarshal balances: %w", err)
	}

	return balances, nil
}

func (c *client) GetOpenOrders(ctx context.Context, market string) ([]Order, error) {
	path := "/open_orders/all/"
	if market != "" {
		path = fmt.Sprintf("/open_orders/%s/", market)
	}

	body, err := c.doPrivateRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}

	var orders []Order
	if err := json.Unmarshal(body, &orders); err != nil {
		return nil, fmt.Errorf("failed to unmarshal orders: %w", err)
	}

	return orders, nil
}

func (c *client) GetOrderStatus(ctx context.Context, orderID string) (*OrderStatus, error) {
	params := url.Values{}
	params.Set("id", orderID)

	body, err := c.doPrivateRequest(ctx, http.MethodPost, "/order_status/", params)
	if err != nil {
		return nil, err
	}

	var status OrderStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order status: %w", err)
	}

	return &status, nil
}

func (c *client) createOrder(ctx context.Context, path string, req CreateOrderRequest) (*CreateOrderResponse, error) {
	params := url.Values{}
	params.Set("amount", req.Amount)

	if req.Price != "" {
		params.Set("price", req.Price)
	}
	if req.LimitPrice != "" {
		params.Set("limit_price", req.LimitPrice)
	}
	if req.DailyOrder {
		params.Set("daily_order", "True")
	}
	if req.IOCOrder {
		params.Set("ioc_order", "True")
	}
	if req.FOKOrder {
		params.Set("fok_order", "True")
	}
	if req.GtdOrder {
		params.Set("gtd_order", "True")
		if req.ExpireTime > 0 {
			params.Set("expire_time", strconv.FormatInt(req.ExpireTime, 10))
		}
	}
	if req.ClientOrderID != "" {
		params.Set("client_order_id", req.ClientOrderID)
	}

	body, err := c.doPrivateRequest(ctx, http.MethodPost, path, params)
	if err != nil {
		return nil, err
	}

	var response CreateOrderResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create order response: %w", err)
	}

	return &response, nil
}

func (c *client) CreateLimitBuyOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error) {
	path := fmt.Sprintf("/buy/%s/", req.Market)
	return c.createOrder(ctx, path, req)
}

func (c *client) CreateLimitSellOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error) {
	path := fmt.Sprintf("/sell/%s/", req.Market)
	return c.createOrder(ctx, path, req)
}

func (c *client) CreateMarketBuyOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error) {
	path := fmt.Sprintf("/buy/market/%s/", req.Market)
	return c.createOrder(ctx, path, req)
}

func (c *client) CreateMarketSellOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error) {
	path := fmt.Sprintf("/sell/market/%s/", req.Market)
	return c.createOrder(ctx, path, req)
}

func (c *client) CancelOrder(ctx context.Context, orderID string) (*CancelOrderResponse, error) {
	params := url.Values{}
	params.Set("id", orderID)

	body, err := c.doPrivateRequest(ctx, http.MethodPost, "/cancel_order/", params)
	if err != nil {
		return nil, err
	}

	var response CancelOrderResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cancel order response: %w", err)
	}

	return &response, nil
}

func (c *client) GetTradingPairs(ctx context.Context) ([]TradingPair, error) {
	body, err := c.doPublicRequest(ctx, "/trading-pairs-info/")
	if err != nil {
		return nil, err
	}

	var pairs []TradingPair
	if err := json.Unmarshal(body, &pairs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trading pairs: %w", err)
	}

	return pairs, nil
}

func (c *client) GetOrderBook(ctx context.Context, market string) (*OrderBookResponse, error) {
	path := fmt.Sprintf("/order_book/%s/", market)

	body, err := c.doPublicRequest(ctx, path)
	if err != nil {
		return nil, err
	}

	var orderBook OrderBookResponse
	if err := json.Unmarshal(body, &orderBook); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order book: %w", err)
	}

	return &orderBook, nil
}

func (c *client) GetTicker(ctx context.Context, market string) (*TickerResponse, error) {
	path := fmt.Sprintf("/ticker/%s/", market)

	body, err := c.doPublicRequest(ctx, path)
	if err != nil {
		return nil, err
	}

	var ticker TickerResponse
	if err := json.Unmarshal(body, &ticker); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ticker: %w", err)
	}

	return &ticker, nil
}

func (c *client) GetOHLC(ctx context.Context, market string, step int, limit int) (*OHLCResponse, error) {
	path := fmt.Sprintf("/ohlc/%s/?step=%d&limit=%d", market, step, limit)

	body, err := c.doPublicRequest(ctx, path)
	if err != nil {
		return nil, err
	}

	var ohlc OHLCResponse
	if err := json.Unmarshal(body, &ohlc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OHLC: %w", err)
	}

	return &ohlc, nil
}
