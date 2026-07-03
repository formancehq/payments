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
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	errorsutils "github.com/formancehq/payments/pkg/domain/errors"
	"github.com/formancehq/payments/pkg/domain/httpwrapper"
	"github.com/formancehq/payments/pkg/domain/metrics"
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

// ClosetimeClose selects the close timestamp for ClosedOrders Start/End
// filtering, so a newly-closed order with an ancient open time still
// falls inside the current window.
const ClosetimeClose = "close"

// DefaultEndpoint is the production base URL. UAT/sandbox is wired by
// setting Config.Endpoint to the VIP host.
const DefaultEndpoint = "https://api.kraken.com"

type client struct {
	httpClient httpwrapper.Client
	endpoint   string
	apiKey     string
	apiSecret  []byte // base64-decoded HMAC secret
}

// New returns a Kraken Pro REST client. The secret is base64-decoded up
// front so a malformed secret fails here, not on the first signed call.
// The transport stays a plain metrics+otel chain; nonce generation and
// signing live in the client (see signRequest) — the signature covers
// the request body, so it must be computed where the body is built, not
// in a body-mutating RoundTripper.
func New(connectorName, apiKey, apiSecret, endpoint string) (Client, error) {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	secret, err := base64.StdEncoding.DecodeString(apiSecret)
	if err != nil {
		return nil, fmt.Errorf("decode apiSecret: %w", err)
	}
	return &client{
		endpoint:  strings.TrimRight(endpoint, "/"),
		apiKey:    apiKey,
		apiSecret: secret,
		httpClient: httpwrapper.NewClient(&httpwrapper.Config{
			// No custom HttpErrorCheckerFn: httpwrapper's default maps
			// 429/4xx/5xx for us. The only Kraken-specific handling is the
			// 200-body error array, parsed below in do().
			Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{
				Transport: otelhttp.NewTransport(http.DefaultTransport),
			}),
		}),
	}, nil
}

// apiError types a Kraken error envelope and routes retriable cases to a
// backoff path (Kraken signals these in the error array, often on HTTP
// 200, so the status code alone won't catch them):
//   - rate-limit/throttle -> ErrStatusCodeTooManyRequests
//   - invalid nonce       -> ErrStatusCodeTooEarly (a stronger backoff,
//     since too many invalid-nonce errors temp-lock the key)
//
// Both map, via the registry, to a retry-with-delay. Fatal-auth and any
// other code stay a bare APIError for the caller to classify.
func apiError(endpoint string, codes []string) error {
	e := &APIError{Endpoint: endpoint, Code: codes[0], All: codes}
	switch {
	case IsRateLimitError(e):
		return errorsutils.NewWrappedError(e, httpwrapper.ErrStatusCodeTooManyRequests)
	case e.Code == invalidNonceCode:
		return errorsutils.NewWrappedError(e, httpwrapper.ErrStatusCodeTooEarly)
	}
	return e
}

// do issues a Kraken REST call and decodes the standard envelope into
// dst. GET hits `/0/public/*` unsigned; POST hits `/0/private/*` and is
// signed by signRequest. errResp captures any 4xx/5xx envelope, which
// Kraken shapes identically to a success body.
func (c *client) do(ctx context.Context, method, uriPath string, params map[string]any, dst any) error {
	var body []byte
	if method == http.MethodPost {
		if params == nil {
			params = map[string]any{}
		}
		raw, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("encode body for %s: %w", uriPath, err)
		}
		body = raw
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+uriPath, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request %s: %w", uriPath, err)
	}
	if method == http.MethodPost {
		c.signRequest(req, uriPath, params)
	}

	var envelope struct {
		Error  []string        `json:"error"`
		Result json.RawMessage `json:"result"`
	}
	var errResp ErrorResponse
	status, err := c.httpClient.Do(ctx, req, &envelope, &errResp)
	if err != nil {
		// httpwrapper returns a non-nil err on non-2xx; surface the
		// envelope's first error code (if any) as a typed APIError so
		// callers can switch on IsFatalAuthError / retriable.
		if len(errResp.Errors) > 0 {
			return apiError(uriPath, errResp.Errors)
		}
		return fmt.Errorf("%s (status %d): %w", uriPath, status, err)
	}
	// Kraken considers a failed or rejected request when only error is defined
	if len(envelope.Error) > 0 && len(envelope.Result) == 0 {
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

// signRequest finalises a private request: it injects a fresh nonce into
// params, re-encodes the body, and sets the api-key/api-nonce/api-sign
// headers. Kept in the client (not the transport) because the signature
// covers the body that carries the nonce. The nonce is a stateless
// strictly-increasing UnixNano: Kraken requires it per key, and the rare
// cross-pod ordering race surfaces as EAPI:Invalid nonce and is retried
// (see apiError / MAPPINGS for the dedicated-key / nonce-window guidance).
func (c *client) signRequest(req *http.Request, uriPath string, params map[string]any) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
	params["nonce"] = nonce
	body, _ := json.Marshal(params) // re-marshal: params was just built from a map
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.apiKey)
	req.Header.Set("api-nonce", nonce)
	req.Header.Set("api-sign", sign(c.apiSecret, uriPath, nonce, body))
}

// sign computes the Kraken API-Sign per docs:
//
//	base64( HMAC-SHA512( secret, uriPath || SHA256(nonce || body) ) )
func sign(secret []byte, uriPath, nonce string, body []byte) string {
	sha := sha256.New()
	sha.Write([]byte(nonce))
	sha.Write(body)
	mac := hmac.New(sha512.New, secret)
	mac.Write([]byte(uriPath))
	mac.Write(sha.Sum(nil))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// GetAssets fetches /0/public/Assets.
func (c *client) GetAssets(ctx context.Context) (map[string]AssetInfo, error) {
	out := make(map[string]AssetInfo)
	if err := c.do(ctx, http.MethodGet, "/0/public/Assets", nil, &out); err != nil {
		return nil, fmt.Errorf("get assets: %w", err)
	}
	return out, nil
}

// GetAssetPairs fetches /0/public/AssetPairs.
func (c *client) GetAssetPairs(ctx context.Context) (map[string]AssetPair, error) {
	out := make(map[string]AssetPair)
	if err := c.do(ctx, http.MethodGet, "/0/public/AssetPairs", nil, &out); err != nil {
		return nil, fmt.Errorf("get asset pairs: %w", err)
	}
	return out, nil
}

// GetBalanceEx fetches /0/private/BalanceEx.
func (c *client) GetBalanceEx(ctx context.Context) (map[string]BalanceExEntry, error) {
	out := make(map[string]BalanceExEntry)
	if err := c.do(ctx, http.MethodPost, "/0/private/BalanceEx", nil, &out); err != nil {
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
	if err := c.do(ctx, http.MethodPost, "/0/private/Ledgers", params, &out); err != nil {
		return LedgersResponse{}, fmt.Errorf("get ledgers: %w", err)
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
	if err := c.do(ctx, http.MethodPost, "/0/private/ClosedOrders", params, &out); err != nil {
		return ClosedOrdersResponse{}, fmt.Errorf("get closed orders: %w", err)
	}
	return out, nil
}
