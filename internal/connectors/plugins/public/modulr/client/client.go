package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/plugins/public/modulr/client/hmac"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetAccounts(ctx context.Context, page, pageSize int, fromCreatedAt time.Time) ([]Account, error)
	GetAccount(ctx context.Context, accountID string) (*Account, error)
	GetBeneficiaries(ctx context.Context, page, pageSize int, modifiedSince time.Time) ([]Beneficiary, error)
	GetPayments(ctx context.Context, paymentType PaymentType, page, pageSize int, modifiedSince time.Time) ([]Payment, error)
	InitiatePayout(ctx context.Context, payoutRequest *PayoutRequest) (*PayoutResponse, error)
	GetPayout(ctx context.Context, payoutID string) (PayoutResponse, error)
	GetTransactions(ctx context.Context, accountID string, page, pageSize int, fromTransactionDate time.Time) ([]Transaction, error)
	InitiateTransfer(ctx context.Context, transferRequest *TransferRequest) (*TransferResponse, error)
	GetTransfer(ctx context.Context, transferID string) (TransferResponse, error)
}

type apiTransport struct {
	apiKey     string
	headers    map[string]string
	underlying http.RoundTripper
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", t.apiKey)

	return t.underlying.RoundTrip(req)
}

type responseWrapper[t any] struct {
	Content    t   `json:"content"`
	Size       int `json:"size"`
	TotalSize  int `json:"totalSize"`
	Page       int `json:"page"`
	TotalPages int `json:"totalPages"`
}

type client struct {
	httpClient httpwrapper.Client
	endpoint   string
}

func (m *client) buildEndpoint(path string, args ...interface{}) string {
	endpoint := strings.TrimSuffix(m.endpoint, "/")
	return fmt.Sprintf("%s/%s", endpoint, fmt.Sprintf(path, args...))
}

const SandboxAPIEndpoint = "https://api-sandbox.modulrfinance.com/api-sandbox-token"

func New(apiKey, apiSecret, endpoint string) (Client, error) {
	if endpoint == "" {
		endpoint = SandboxAPIEndpoint
	}

	headers, err := hmac.GenerateHeaders(apiKey, apiSecret, "", false)
	if err != nil {
		return nil, fmt.Errorf("failed to generate headers: %w", err)
	}
	config := &httpwrapper.Config{
		CommonMetricsAttributes: httpwrapper.CommonMetricsAttributesFor("modulr"),
		Transport: &apiTransport{
			headers:    headers,
			apiKey:     apiKey,
			underlying: otelhttp.NewTransport(http.DefaultTransport),
		},
	}

	return &client{
		httpClient: httpwrapper.NewClient(config),
		endpoint:   endpoint,
	}, nil
}

type ErrorResponse struct {
	Field         string `json:"field"`
	Code          string `json:"code"`
	Message       string `json:"message"`
	ErrorCode     string `json:"errorCode"`
	SourceService string `json:"sourceService"`
}
