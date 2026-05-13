// Package client is the hand-rolled HTTP client used by the Universal CE
// Connector to talk to a counterparty implementing the universal-openapi.yaml
// v1 contract. Every operation is wrapped through httpwrapper so we get OTel
// spans, default-transport TLS verification, the standard error-status
// mapping, and metrics tagged by the "universal" connector name.
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

// Client is the contract surface every per-primitive method depends on.
// Hand-rolled to keep the universal package free of generated code (no
// go.mod replace directive) and to map cleanly onto the operations
// described in contract/universal-openapi.yaml.
type Client interface {
	// SetIdempotencyHeader switches the header name used on every
	// mutating POST. Called once from plugin.Install when the
	// counterparty advertises features.idempotencyHeader.
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

// Pagination carries everything a paginated list endpoint needs. The
// counterparty announces which knobs it supports in /v1/capabilities; we send
// all of them and let the server pick what it cares about.
type Pagination struct {
	Cursor        string
	PageNumber    int
	PageSize      int
	UpdatedAtFrom time.Time
}

// IdempotencyHeader is the canonical header name we use on every mutating
// POST. Counterparties can declare an alternative via /v1/capabilities's
// features.idempotencyHeader; the plugin sends both the canonical name and
// the announced override when they differ.
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

// authTransport is intentionally tiny: bearer-only, no logging of the token.
// Constant-time isn't relevant on this side (we control the secret), and we
// rely on the default http.Transport for TLS verification.
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

func New(connectorName, endpoint, apiKey string) Client {
	return &client{
		httpClient: httpwrapper.NewClient(&httpwrapper.Config{
			Timeout: defaultTimeout,
			Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{
				Transport: &authTransport{apiKey: apiKey, underlying: http.DefaultTransport},
			}),
		}),
		endpoint:          strings.TrimSuffix(endpoint, "/"),
		apiKey:            apiKey,
		idempotencyHeader: IdempotencyHeader,
	}
}

// SetIdempotencyHeader overrides the header name used on POST requests. The
// plugin calls this once after install if /v1/capabilities returned a
// non-empty features.idempotencyHeader override.
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

// do issues a request and unmarshals into out. The error envelope honors both
// RFC 7807 (application/problem+json) and the legacy {message, errors[]}
// shape via *Error's UnmarshalJSON. Successful no-content responses are
// treated as success regardless of out being nil.
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
		// Always send the canonical name too, so counterparties that use the
		// default still dedup correctly when an override was declared.
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
