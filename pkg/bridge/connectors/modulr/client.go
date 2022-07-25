package modulr

import (
	"fmt"
	"net/http"

	"github.com/numary/payments/pkg/bridge/connectors/modulr/hmac"
)

const (
	apiEndpoint = "https://api-sandbox.modulrfinance.com/api-sandbox-token"
)

type Credentials struct {
	APIKey    string `json:"api_key" bson:"api_key"`
	APISecret string `json:"api_secret" bson:"api_secret"`
}

type apiTransport struct {
	credentials Credentials
	headers     map[string]string
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// for k, v := range t.headers {
	// 	req.Header.Add(k, v)
	// }

	req.Header.Add("Authorization", t.credentials.APIKey)

	return http.DefaultTransport.RoundTrip(req)
}

type ResponseWrapper[T any] struct {
	Content    T   `json:"content"`
	Size       int `json:"size"`
	TotalSize  int `json:"totalSize"`
	Page       int `json:"page"`
	TotalPages int `json:"totalPages"`
}

type ModulrClient struct {
	httpClient *http.Client
}

func (m *ModulrClient) Endpoint(path string, args ...interface{}) string {
	return fmt.Sprintf("%s/%s", apiEndpoint, fmt.Sprintf(path, args...))
}

func NewModulrClient(c Credentials) *ModulrClient {
	headers, _ := hmac.GenerateHeaders(c.APIKey, c.APIKey, "", false)

	return &ModulrClient{
		httpClient: &http.Client{
			Transport: &apiTransport{
				headers:     headers,
				credentials: c,
			},
		},
	}
}
