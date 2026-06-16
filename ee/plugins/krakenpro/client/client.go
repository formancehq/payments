package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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
}

// New returns a Kraken Pro REST client. The secret is base64-decoded up
// front so an invalid secret fails here, not on the first signed call.
// Signing is handled by signingTransport, slotted innermost (closest to
// the wire) so it signs over the exact bytes that go out.
func New(connectorName, apiKey, apiSecret, endpoint string) (Client, error) {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	secret, err := base64.StdEncoding.DecodeString(apiSecret)
	if err != nil {
		return nil, fmt.Errorf("decode apiSecret: %w", err)
	}
	signing := newSigningTransport(apiKey, secret, http.DefaultTransport)
	return &client{
		endpoint: strings.TrimRight(endpoint, "/"),
		httpClient: httpwrapper.NewClient(&httpwrapper.Config{
			Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{
				Transport: otelhttp.NewTransport(signing),
			}),
		}),
	}, nil
}

// apiError types a Kraken error envelope. Rate-limit and (cross-pod)
// invalid-nonce codes are wrapped as httpwrapper.ErrStatusCodeTooManyRequests
// (the registry maps that to plugins.ErrUpstreamRatelimit -> retry with
// backoff) because Kraken signals both in the error array, often on
// HTTP 200 — the status code alone won't catch them.
func apiError(endpoint string, codes []string) error {
	e := &APIError{Endpoint: endpoint, Code: codes[0], All: codes}
	if IsRetriableError(e) {
		return errorsutils.NewWrappedError(e, httpwrapper.ErrStatusCodeTooManyRequests)
	}
	return e
}

// do issues a Kraken REST call and decodes the standard envelope into
// dst. GET hits `/0/public/*` unsigned; POST hits `/0/private/*` and is
// signed by signingTransport. errResp captures any 4xx/5xx envelope,
// which Kraken shapes identically to a success body.
func (c *client) do(ctx context.Context, method, uriPath string, params map[string]any, dst any) error {
	var body io.Reader
	if method == http.MethodPost {
		raw, err := json.Marshal(orEmptyParams(params))
		if err != nil {
			return fmt.Errorf("encode body for %s: %w", uriPath, err)
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+uriPath, body)
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
		// httpwrapper returns a non-nil err on non-2xx; surface the
		// envelope's first error code (if any) as a typed APIError so
		// callers can switch on IsFatalAuthError / retriable.
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

// orEmptyParams guarantees a non-nil body so even param-less private
// calls (e.g. BalanceEx) send `{}` for the transport to inject the nonce.
func orEmptyParams(params map[string]any) map[string]any {
	if params == nil {
		return map[string]any{}
	}
	return params
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
	if err := c.do(ctx, http.MethodPost, "/0/private/OpenOrders", params, &out); err != nil {
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
	if err := c.do(ctx, http.MethodPost, "/0/private/ClosedOrders", params, &out); err != nil {
		return ClosedOrdersResponse{}, fmt.Errorf("get closed orders: %w", err)
	}
	return out, nil
}
