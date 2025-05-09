package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetOrganization(ctx context.Context) (*Organization, error)
	GetBeneficiaries(ctx context.Context, updatedAtFrom time.Time, pageSize int) ([]Beneficiary, error)
	GetTransactions(ctx context.Context, bankAccountId string, updatedAtFrom time.Time, transactionStatusToFetch string, pageSize int) ([]Transactions, error)
}

type client struct {
	httpClient httpwrapper.Client

	clientID     string
	apiKey       string
	endpoint     string
	stagingToken string
}

type apiTransport struct {
	client     *client
	underlying http.RoundTripper
}

type MetaPagination struct {
	CurrentPage int `json:"current_page"`
	PrevPage    any `json:"prev_page"`
	NextPage    any `json:"next_page"`
	TotalPages  int `json:"total_pages"`
	PerPage     int `json:"per_page"`
	TotalCount  int `json:"total_count"`
}

const QONTO_TIMEFORMAT = "2006-01-02T15:04:05.999Z"

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	auth := fmt.Sprintf("%s:%s", t.client.clientID, t.client.apiKey)

	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")
	if t.client.stagingToken != "" {
		req.Header.Set("X-Qonto-Staging-Token", t.client.stagingToken)
	}
	return t.underlying.RoundTrip(req)
}

func New(connectorName, clientID, apiKey, endpoint, stagingToken string) Client {
	endpoint = strings.TrimSuffix(endpoint, "/")

	client := &client{
		clientID:     clientID,
		apiKey:       apiKey,
		endpoint:     endpoint,
		stagingToken: stagingToken,
	}

	apiTransport := &apiTransport{
		client:     client,
		underlying: metrics.NewTransport(connectorName, metrics.TransportOpts{}),
	}
	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{
			Transport: apiTransport,
		}),
	}
	client.httpClient = httpwrapper.NewClient(config)

	return client
}

func (c *client) buildEndpoint(path string, args ...interface{}) string {
	return fmt.Sprintf("%s/%s", c.endpoint, fmt.Sprintf(path, args...))
}
