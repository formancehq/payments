// Package client is the HTTP client for the universal-openapi.yaml v1
// contract. Wrapped through httpwrapper for OTel spans, retryable
// status-code mapping, and per-connector metrics.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client

// Client mirrors contract/universal-openapi.yaml. Hand-rolled (no
// generated code) so the universal package needs no go.mod replace.
type Client interface {
	// SetIdempotencyHeader overrides the canonical header name on
	// mutating POSTs. Called from plugin.Install when the counterparty
	// advertises features.idempotencyHeader.
	SetIdempotencyHeader(name string)

	GetCapabilities(ctx context.Context) (*CapabilitiesResponse, error)

	ListAccounts(ctx context.Context, page Pagination) (*AccountsPage, error)
	ListExternalAccounts(ctx context.Context, page Pagination) (*AccountsPage, error)
	GetBalances(ctx context.Context, accountID string) (*BalancesResponse, error)

	ListPayments(ctx context.Context, page Pagination) (*PaymentsPage, error)
	ListOrders(ctx context.Context, page Pagination) (*OrdersPage, error)
	ListConversions(ctx context.Context, page Pagination) (*ConversionsPage, error)
	ListOthers(ctx context.Context, name string, page Pagination) (*OthersPage, error)

	CreatePayout(ctx context.Context, idemKey string, req *PayoutRequest) (*PayoutResponse, error)
	GetPayout(ctx context.Context, id string) (*PayoutResponse, error)
	ReversePayout(ctx context.Context, idemKey string, id string, req *ReverseRequest) (*PayoutResponse, error)

	CreateTransfer(ctx context.Context, idemKey string, req *TransferRequest) (*TransferResponse, error)
	GetTransfer(ctx context.Context, id string) (*TransferResponse, error)
	ReverseTransfer(ctx context.Context, idemKey string, id string, req *ReverseRequest) (*TransferResponse, error)

	CreateBankAccount(ctx context.Context, idemKey string, req *BankAccountRequest) (*BankAccountResponse, error)

	CreateWebhookSubscription(ctx context.Context, idemKey string, req *WebhookSubscriptionRequest) (*WebhookSubscriptionResponse, error)
	DeleteWebhookSubscription(ctx context.Context, id string) error
}

// Pagination carries every paginated list knob. We send all of them on
// every request; the counterparty picks what it cares about per its
// features.pagination advertisement.
type Pagination struct {
	Cursor        string
	PageNumber    int
	PageSize      int
	UpdatedAtFrom time.Time
}

// IdempotencyHeader is the canonical header on mutating POSTs. When the
// counterparty advertises an alias via features.idempotencyHeader, do()
// sends BOTH so retries dedup either way.
const (
	IdempotencyHeader = "Idempotency-Key"
	defaultTimeout    = 30 * time.Second
)

type client struct {
	httpClient        httpwrapper.Client
	endpoint          string
	apiKey            string
	idempotencyHeader string
}

// authTransport stamps the bearer token. Token is never logged.
type authTransport struct {
	apiKey     string
	underlying http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	return t.underlying.RoundTrip(req)
}

// pooledTransport clones http.DefaultTransport and lifts the per-host
// connection cap. The stdlib default (`MaxIdleConnsPerHost = 2`) is
// fine for browsers and rare-call SDKs; a Formance connector
// consistently hammers a single counterparty host from many Temporal
// activities in parallel, so 2 idle conns leads to constant TCP/TLS
// re-handshake churn at 20k tx/day and up.
func pooledTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 200
	t.MaxIdleConnsPerHost = 64
	t.MaxConnsPerHost = 0 // unbounded; pooled idle is the throttle
	t.IdleConnTimeout = 90 * time.Second
	return t
}

// New constructs a Client. httpwrapper.NewClient wraps the transport
// once in otelhttp; do not pre-wrap here.
func New(connectorName, endpoint, apiKey string) Client {
	return &client{
		httpClient: httpwrapper.NewClient(&httpwrapper.Config{
			Timeout: defaultTimeout,
			Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{
				Transport: &authTransport{apiKey: apiKey, underlying: pooledTransport()},
			}),
		}),
		endpoint:          strings.TrimSuffix(endpoint, "/"),
		apiKey:            apiKey,
		idempotencyHeader: IdempotencyHeader,
	}
}

func (c *client) SetIdempotencyHeader(name string) {
	if name == "" {
		return
	}
	c.idempotencyHeader = name
}

func (c *client) url(path string) string { return c.endpoint + path }

func (c *client) addPagination(u string, p Pagination) string {
	if p.Cursor == "" && p.PageNumber == 0 && p.PageSize == 0 && p.UpdatedAtFrom.IsZero() {
		return u
	}
	q := url.Values{}
	if p.Cursor != "" {
		q.Set("cursor", p.Cursor)
	}
	if p.PageNumber > 0 {
		q.Set("page", fmt.Sprintf("%d", p.PageNumber))
	}
	if p.PageSize > 0 {
		q.Set("pageSize", fmt.Sprintf("%d", p.PageSize))
	}
	if !p.UpdatedAtFrom.IsZero() {
		q.Set("updatedAtFrom", p.UpdatedAtFrom.UTC().Format(time.RFC3339Nano))
	}
	return u + "?" + q.Encode()
}

// do issues a request and unmarshals into out. The error envelope
// accepts both RFC 7807 and the legacy `{message, errors}` shape — see
// Error.
func (c *client) do(ctx context.Context, method, url string, idemKey string, body, out any) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshalling request body for %s %s: %w", method, url, err)
		}
		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return fmt.Errorf("building %s %s request: %w", method, url, err)
	}
	if reader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if idemKey != "" {
		req.Header.Set(c.idempotencyHeader, idemKey)
		if c.idempotencyHeader != IdempotencyHeader {
			req.Header.Set(IdempotencyHeader, idemKey)
		}
	}

	apiErr := &Error{}
	status, err := c.httpClient.Do(ctx, req, out, apiErr)
	if err == nil {
		return nil
	}
	apiErr.HTTPStatus = status
	apiErr.Underlying = err
	return apiErr
}
