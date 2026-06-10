package client

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client

// Client is the read-only Kraken Pro REST surface used by the connector.
// Each method maps 1:1 to a Kraken endpoint. The implementation handles
// HMAC-SHA512 signing for /private/* paths.
type Client interface {
	GetAssets(ctx context.Context) (map[string]AssetInfo, error)
	GetAssetPairs(ctx context.Context) (map[string]AssetPair, error)
	GetBalanceEx(ctx context.Context) (map[string]BalanceExEntry, error)
	GetLedgers(ctx context.Context, params LedgersParams) (LedgersResponse, error)
	GetOpenOrders(ctx context.Context, params OpenOrdersParams) (OpenOrdersResponse, error)
	GetClosedOrders(ctx context.Context, params ClosedOrdersParams) (ClosedOrdersResponse, error)
}

// LedgersParams filters /0/private/Ledgers. Pagination is a frozen
// window + ofs walk driven by ledgerWindow (state.go).
type LedgersParams struct {
	Start        float64 // exclusive lower-bound timestamp (committed watermark)
	End          float64 // inclusive upper-bound timestamp, frozen at window start
	Type         string  // "all" / "deposit" / "withdrawal" / "conversion" / ...
	Offset       int     // ofs page position within the frozen window
	WithoutCount bool
}

// OpenOrdersParams filters /0/private/OpenOrders. Cursor pagination is
// OpenOrders-only (ClosedOrders has none); Cursor carries the prior
// response's cursor.next token.
type OpenOrdersParams struct {
	Trades     bool
	Cursor     string
	WithCursor bool
	Limit      int
	Userref    int
}

// ClosedOrdersParams filters /0/private/ClosedOrders. Same frozen
// window + ofs pagination as Ledgers (no cursor).
type ClosedOrdersParams struct {
	Trades       bool
	Start        float64
	End          float64
	Offset       int
	Closetime    string // which timestamp Start/End apply to: "open" | "close" | "both"
	WithoutCount bool
}

// DefaultEndpoint is the production base URL. UAT/sandbox is wired by
// setting Config.Endpoint to the VIP host.
const DefaultEndpoint = "https://api.kraken.com"

type client struct {
	httpClient httpwrapper.Client
	endpoint   string
	apiKey     string
	apiSecret  []byte       // base64-decoded
	nonce      atomic.Int64 // strictly-monotonic per-key nonce; see nextNonce
}

// New returns a Kraken Pro REST client. The secret is base64-decoded up
// front so an invalid secret fails here, not on the first signed call.
func New(connectorName, apiKey, apiSecret, endpoint string) (Client, error) {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	secret, err := base64.StdEncoding.DecodeString(apiSecret)
	if err != nil {
		return nil, fmt.Errorf("decode apiSecret: %w", err)
	}
	c := &client{
		endpoint:  strings.TrimRight(endpoint, "/"),
		apiKey:    apiKey,
		apiSecret: secret,
	}
	// Seed with UnixNano so the nonce starts above any prior ms/us-precision
	// caller: Kraken rejects any nonce <= the highest ever used for the key.
	c.nonce.Store(time.Now().UnixNano())
	c.httpClient = httpwrapper.NewClient(&httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}),
	})
	return c, nil
}

// nextNonce returns max(prev+1, now) as an ASCII nonce, so a backward
// clock tick (NTP) can't violate the strictly-increasing invariant
// Kraken enforces per API key.
func (c *client) nextNonce() string {
	for {
		prev := c.nonce.Load()
		now := time.Now().UnixNano()
		next := prev + 1
		if now > next {
			next = now
		}
		if c.nonce.CompareAndSwap(prev, next) {
			return strconv.FormatInt(next, 10)
		}
	}
}

// signPath signs the per-request inputs per Kraken docs:
//
//	API-Sign = base64( HMAC-SHA512( secret, uriPath || SHA256(nonceASCII || body) ) )
//
// uriPath is everything after the host (e.g. "/0/private/Balance").
func (c *client) signPath(uriPath, nonce string, body []byte) string {
	sha := sha256.New()
	sha.Write([]byte(nonce))
	sha.Write(body)
	mac := hmac.New(sha512.New, c.apiSecret)
	mac.Write([]byte(uriPath))
	mac.Write(sha.Sum(nil))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// doPublic issues an unsigned GET to a /public/* endpoint, decodes
// the standard Kraken envelope, and unmarshals result into dst.
func (c *client) doPublic(ctx context.Context, uriPath string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+uriPath, nil)
	if err != nil {
		return fmt.Errorf("build request %s: %w", uriPath, err)
	}
	var envelope struct {
		Error  []string        `json:"error"`
		Result json.RawMessage `json:"result"`
	}
	var errResp ErrorResponse
	status, err := c.httpClient.Do(ctx, req, &envelope, &errResp)
	if err != nil {
		return fmt.Errorf("%s (status %d): %w", uriPath, status, err)
	}
	if len(envelope.Error) > 0 {
		return apiError(uriPath, envelope.Error)
	}
	if dst == nil || len(envelope.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(envelope.Result, dst); err != nil {
		return fmt.Errorf("decode %s result: %w", uriPath, err)
	}
	return nil
}

// apiError types a Kraken error envelope. Rate-limit codes are wrapped
// as httpwrapper.ErrStatusCodeTooManyRequests (the registry maps that to
// plugins.ErrUpstreamRatelimit) because Kraken signals throttling in the
// error array, often on HTTP 200 — the status code alone won't catch it.
func apiError(endpoint string, codes []string) error {
	e := &APIError{Endpoint: endpoint, Code: codes[0], All: codes}
	if IsRateLimitError(e) {
		return errorsutils.NewWrappedError(e, httpwrapper.ErrStatusCodeTooManyRequests)
	}
	return e
}

// doPrivate issues a signed POST to a /private/* endpoint with a
// JSON body that always carries the nonce. Result is decoded into
// dst. errResp captures any 4xx/5xx envelope shape.
func (c *client) doPrivate(ctx context.Context, uriPath string, params map[string]any, dst any) error {
	nonce := c.nextNonce()
	if params == nil {
		params = make(map[string]any, 1)
	}
	params["nonce"] = nonce

	body, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("encode body for %s: %w", uriPath, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+uriPath, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request %s: %w", uriPath, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.apiKey)
	req.Header.Set("api-nonce", nonce)
	req.Header.Set("api-sign", c.signPath(uriPath, nonce, body))

	var envelope struct {
		Error  []string        `json:"error"`
		Result json.RawMessage `json:"result"`
	}
	var errResp ErrorResponse
	status, err := c.httpClient.Do(ctx, req, &envelope, &errResp)
	if err != nil {
		// httpwrapper returns a non-nil err on non-2xx; surface the
		// envelope's first error code (if any) as a typed APIError
		// so callers can switch on IsFatalAuthError / rate limit.
		if len(errResp.Errors) > 0 {
			return apiError(uriPath, errResp.Errors)
		}
		return fmt.Errorf("%s (status %d): %w", uriPath, status, err)
	}
	if len(envelope.Error) > 0 {
		return apiError(uriPath, envelope.Error)
	}
	if dst == nil || len(envelope.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(envelope.Result, dst); err != nil {
		return fmt.Errorf("decode %s result: %w", uriPath, err)
	}
	return nil
}

// GetAssets fetches /0/public/Assets.
func (c *client) GetAssets(ctx context.Context) (map[string]AssetInfo, error) {
	out := make(map[string]AssetInfo)
	if err := c.doPublic(ctx, "/0/public/Assets", &out); err != nil {
		return nil, fmt.Errorf("get assets: %w", err)
	}
	return out, nil
}

// GetAssetPairs fetches /0/public/AssetPairs.
func (c *client) GetAssetPairs(ctx context.Context) (map[string]AssetPair, error) {
	out := make(map[string]AssetPair)
	if err := c.doPublic(ctx, "/0/public/AssetPairs", &out); err != nil {
		return nil, fmt.Errorf("get asset pairs: %w", err)
	}
	return out, nil
}

// GetBalanceEx fetches /0/private/BalanceEx.
func (c *client) GetBalanceEx(ctx context.Context) (map[string]BalanceExEntry, error) {
	out := make(map[string]BalanceExEntry)
	if err := c.doPrivate(ctx, "/0/private/BalanceEx", nil, &out); err != nil {
		return nil, fmt.Errorf("get balance ex: %w", err)
	}
	return out, nil
}

// GetLedgers fetches /0/private/Ledgers.
func (c *client) GetLedgers(ctx context.Context, p LedgersParams) (LedgersResponse, error) {
	params := map[string]any{}
	if p.Start > 0 {
		params["start"] = p.Start
	}
	if p.End > 0 {
		params["end"] = p.End
	}
	if p.Type != "" {
		params["type"] = p.Type
	}
	if p.Offset > 0 {
		params["ofs"] = p.Offset
	}
	if p.WithoutCount {
		params["without_count"] = true
	}
	var out LedgersResponse
	if err := c.doPrivate(ctx, "/0/private/Ledgers", params, &out); err != nil {
		return LedgersResponse{}, fmt.Errorf("get ledgers: %w", err)
	}
	return out, nil
}

// GetOpenOrders fetches /0/private/OpenOrders. Trades=true makes each
// row carry its per-fill txids inline, avoiding a separate QueryTrades
// call. WithCursor/Cursor/Limit drive the cursor drain.
func (c *client) GetOpenOrders(ctx context.Context, p OpenOrdersParams) (OpenOrdersResponse, error) {
	params := map[string]any{}
	if p.Trades {
		params["trades"] = true
	}
	if p.WithCursor {
		params["with_cursor"] = true
	}
	if p.Cursor != "" {
		params["cursor"] = p.Cursor
	}
	if p.Limit > 0 {
		params["limit"] = p.Limit
	}
	if p.Userref != 0 {
		params["userref"] = p.Userref
	}
	var out OpenOrdersResponse
	if err := c.doPrivate(ctx, "/0/private/OpenOrders", params, &out); err != nil {
		return OpenOrdersResponse{}, fmt.Errorf("get open orders: %w", err)
	}
	return out, nil
}

// GetClosedOrders fetches /0/private/ClosedOrders. No cursor support, so
// it uses the frozen-window + ofs walk on Closetime="close" — that way a
// newly-closed order with an ancient opentm still falls in the window.
func (c *client) GetClosedOrders(ctx context.Context, p ClosedOrdersParams) (ClosedOrdersResponse, error) {
	params := map[string]any{}
	if p.Trades {
		params["trades"] = true
	}
	if p.Start > 0 {
		params["start"] = p.Start
	}
	if p.End > 0 {
		params["end"] = p.End
	}
	if p.Offset > 0 {
		params["ofs"] = p.Offset
	}
	if p.Closetime != "" {
		params["closetime"] = p.Closetime
	}
	if p.WithoutCount {
		params["without_count"] = true
	}
	var out ClosedOrdersResponse
	if err := c.doPrivate(ctx, "/0/private/ClosedOrders", params, &out); err != nil {
		return ClosedOrdersResponse{}, fmt.Errorf("get closed orders: %w", err)
	}
	return out, nil
}
