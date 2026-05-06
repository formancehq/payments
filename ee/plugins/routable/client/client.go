package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// DefaultBaseURL is the production Routable API endpoint. Sandbox is
// https://api.sandbox.routable.com and is selected via the connector config.
const DefaultBaseURL = "https://api.routable.com"

// Client is the minimal Routable API surface the dedicated EE plugin needs:
// fetching settings accounts (for INTERNAL accounts and balances), companies
// (for EXTERNAL accounts), payables and receivables (for payment sync), and
// creating + reading payables (for transfer/payout initiation and polling).
//
//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	ListAccounts(ctx context.Context, page, pageSize int) (*ListAccountsResponse, error)
	GetAccount(ctx context.Context, id string) (*Account, error)

	ListCompanies(ctx context.Context, page, pageSize int) (*ListCompaniesResponse, error)

	ListPayables(ctx context.Context, page, pageSize int, statusChangedAtGte time.Time) (*ListPayablesResponse, error)
	GetPayable(ctx context.Context, id string) (*Payable, error)
	CreatePayable(ctx context.Context, req CreatePayableRequest) (*Payable, error)

	ListReceivables(ctx context.Context, page, pageSize int, statusChangedAtGte time.Time) (*ListReceivablesResponse, error)
}

type client struct {
	httpClient httpwrapper.Client
	endpoint   string
}

type apiTransport struct {
	apiKey     string
	underlying http.RoundTripper
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Accept", "application/json")
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return t.underlying.RoundTrip(req)
}

// New builds a Routable client wired with the standard Formance HTTP wrapper
// (otel + metrics + error mapping). connectorName is used as the metrics
// label so per-connector dashboards continue to work unchanged.
func New(connectorName, apiKey, endpoint string) Client {
	endpoint = strings.TrimSuffix(endpoint, "/")
	if endpoint == "" {
		endpoint = DefaultBaseURL
	}

	transport := &apiTransport{
		apiKey:     apiKey,
		underlying: otelhttp.NewTransport(http.DefaultTransport),
	}

	return &client{
		endpoint: endpoint,
		httpClient: httpwrapper.NewClient(&httpwrapper.Config{
			Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{Transport: transport}),
		}),
	}
}

func (c *client) buildURL(path string, query url.Values) string {
	u := c.endpoint + path
	if len(query) == 0 {
		return u
	}
	return u + "?" + query.Encode()
}

func paginationQuery(page, pageSize int) url.Values {
	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}
	return q
}

// do executes a request and unmarshals the success body or surfaces the
// Routable error envelope. The httpwrapper maps 4xx/5xx to typed sentinels
// (ErrStatusCodeClientError/ServerError/TooManyRequests) which callers can
// classify with errors.Is.
func (c *client) do(ctx context.Context, method, path string, query url.Values, body any, idempotencyKey string, out any) (int, error) {
	reqBody := io.Reader(http.NoBody)
	if body != nil {
		buf := bytes.NewBuffer(nil)
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return 0, fmt.Errorf("encoding request body: %w", err)
		}
		reqBody = buf
	}

	req, err := http.NewRequestWithContext(ctx, method, c.buildURL(path, query), reqBody)
	if err != nil {
		return 0, fmt.Errorf("creating request: %w", err)
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	var apiErr ErrorResponse
	statusCode, doErr := c.httpClient.Do(ctx, req, out, &apiErr)
	if doErr != nil {
		return statusCode, fmt.Errorf("%w: %s", doErr, apiErr.Error())
	}
	return statusCode, nil
}

func (c *client) ListAccounts(ctx context.Context, page, pageSize int) (*ListAccountsResponse, error) {
	var resp ListAccountsResponse
	if _, err := c.do(ctx, http.MethodGet, "/v1/settings/accounts", paginationQuery(page, pageSize), nil, "", &resp); err != nil {
		return nil, fmt.Errorf("listing accounts: %w", err)
	}
	return &resp, nil
}

func (c *client) GetAccount(ctx context.Context, id string) (*Account, error) {
	if id == "" {
		return nil, errors.New("account id is required")
	}
	var resp Account
	if _, err := c.do(ctx, http.MethodGet, "/v1/settings/accounts/"+url.PathEscape(id), nil, nil, "", &resp); err != nil {
		return nil, fmt.Errorf("getting account %s: %w", id, err)
	}
	return &resp, nil
}

func (c *client) ListCompanies(ctx context.Context, page, pageSize int) (*ListCompaniesResponse, error) {
	var resp ListCompaniesResponse
	if _, err := c.do(ctx, http.MethodGet, "/v1/companies", paginationQuery(page, pageSize), nil, "", &resp); err != nil {
		return nil, fmt.Errorf("listing companies: %w", err)
	}
	return &resp, nil
}

func (c *client) ListPayables(ctx context.Context, page, pageSize int, statusChangedAtGte time.Time) (*ListPayablesResponse, error) {
	q := paginationQuery(page, pageSize)
	if !statusChangedAtGte.IsZero() {
		q.Set("status_changed_at.gte", statusChangedAtGte.UTC().Format(time.RFC3339))
	}
	var resp ListPayablesResponse
	if _, err := c.do(ctx, http.MethodGet, "/v1/payables", q, nil, "", &resp); err != nil {
		return nil, fmt.Errorf("listing payables: %w", err)
	}
	return &resp, nil
}

func (c *client) GetPayable(ctx context.Context, id string) (*Payable, error) {
	if id == "" {
		return nil, errors.New("payable id is required")
	}
	var resp Payable
	statusCode, err := c.do(ctx, http.MethodGet, "/v1/payables/"+url.PathEscape(id), nil, nil, "", &resp)
	if err != nil {
		if statusCode == http.StatusNotFound {
			return nil, ErrPayableNotFound
		}
		return nil, fmt.Errorf("getting payable %s: %w", id, err)
	}
	return &resp, nil
}

func (c *client) CreatePayable(ctx context.Context, req CreatePayableRequest) (*Payable, error) {
	if err := validateCreatePayable(req); err != nil {
		return nil, err
	}
	var resp Payable
	if _, err := c.do(ctx, http.MethodPost, "/v1/payables", nil, req, req.IdempotencyKey, &resp); err != nil {
		return nil, fmt.Errorf("creating payable: %w", err)
	}
	return &resp, nil
}

func validateCreatePayable(req CreatePayableRequest) error {
	switch {
	case req.Type == "":
		return errors.New("create payable: type is required")
	case req.DeliveryMethod == "":
		return errors.New("create payable: delivery_method is required")
	case req.PayToCompany == "":
		return errors.New("create payable: pay_to_company is required")
	case req.WithdrawFromAccount == "":
		return errors.New("create payable: withdraw_from_account is required")
	case req.Amount == "":
		return errors.New("create payable: amount is required")
	case len(req.LineItems) == 0:
		return errors.New("create payable: at least one line item is required")
	case req.ActingTeamMember == "":
		return errors.New("create payable: acting_team_member is required")
	}
	return nil
}

func (c *client) ListReceivables(ctx context.Context, page, pageSize int, statusChangedAtGte time.Time) (*ListReceivablesResponse, error) {
	q := paginationQuery(page, pageSize)
	if !statusChangedAtGte.IsZero() {
		q.Set("status_changed_at.gte", statusChangedAtGte.UTC().Format(time.RFC3339))
	}
	var resp ListReceivablesResponse
	if _, err := c.do(ctx, http.MethodGet, "/v1/receivables", q, nil, "", &resp); err != nil {
		return nil, fmt.Errorf("listing receivables: %w", err)
	}
	return &resp, nil
}
