package client

import (
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/pkg/connectors/modulr/hmac"
)

type apiTransport struct {
	apiKey  string
	headers map[string]string
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", t.apiKey)

	return http.DefaultTransport.RoundTrip(req)
}

type responseWrapper[t any] struct {
	Content    t   `json:"content"`
	Size       int `json:"size"`
	TotalSize  int `json:"totalSize"`
	Page       int `json:"page"`
	TotalPages int `json:"totalPages"`
}

type Client struct {
	httpClient *http.Client
	endpoint   string
}

func (m *Client) buildEndpoint(path string, args ...interface{}) string {
	return fmt.Sprintf("%s/%s", m.endpoint, fmt.Sprintf(path, args...))
}

const sandboxAPIEndpoint = "https://api-sandbox.modulrfinance.com/api-sandbox-token"

func NewClient(apiKey, apiSecret, endpoint string) (*Client, error) {
	if endpoint == "" {
		endpoint = sandboxAPIEndpoint
	}

	headers, err := hmac.GenerateHeaders(apiKey, apiSecret, "", false)
	if err != nil {
		return nil, fmt.Errorf("failed to generate headers: %w", err)
	}

	return &Client{
		httpClient: &http.Client{
			Transport: &apiTransport{
				headers: headers,
				apiKey:  apiKey,
			},
		},
		endpoint: endpoint,
	}, nil
}
