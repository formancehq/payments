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
	GetBeneficiaries(ctx context.Context, updatedAtFrom time.Time, page, pageSize int) ([]Beneficiary, error)
	GetTransactions(ctx context.Context, bankAccountId string, updatedAtFrom time.Time, transactionStatusToFetch string, page, pageSize int) ([]Transactions, error)
}

type client struct {
	httpClient httpwrapper.Client
	endpoint   string
}

type apiTransport struct {
	clientID     string
	underlying   http.RoundTripper
	apiKey       string
	stagingToken string
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
const QONTO_MAX_PAGE_SIZE = 100

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	auth := fmt.Sprintf("%s:%s", t.clientID, t.apiKey)

	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")
	if t.stagingToken != "" {
		req.Header.Set("X-Qonto-Staging-Token", t.stagingToken)
	}
	return t.underlying.RoundTrip(req)
}

func New(connectorName, clientID, apiKey, endpoint, stagingToken string) Client {
	endpoint = strings.TrimSuffix(endpoint, "/")

	client := &client{
		endpoint: endpoint,
	}

	apiTransport := &apiTransport{
		clientID:     clientID,
		apiKey:       apiKey,
		stagingToken: stagingToken,
		underlying:   http.DefaultTransport,
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
