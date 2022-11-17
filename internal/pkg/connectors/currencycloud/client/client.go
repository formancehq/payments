package client

import (
	"context"
	"fmt"
	"net/http"
)

type apiTransport struct {
	authToken string
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("X-Auth-Token", t.authToken)

	return http.DefaultTransport.RoundTrip(req)
}

type Client struct {
	httpClient *http.Client
	endpoint   string
	loginID    string
	apiKey     string
}

func (c *Client) buildEndpoint(path string, args ...interface{}) string {
	return fmt.Sprintf("%s/%s", c.endpoint, fmt.Sprintf(path, args...))
}

const devAPIEndpoint = "https://devapi.currencycloud.com"

// NewClient creates a new client for the CurrencyCloud API.
func NewClient(ctx context.Context, loginID, apiKey, endpoint string) (*Client, error) {
	if endpoint == "" {
		endpoint = devAPIEndpoint
	}

	c := &Client{
		httpClient: &http.Client{},
		endpoint:   endpoint,
		loginID:    loginID,
		apiKey:     apiKey,
	}

	authToken, err := c.authenticate(ctx)
	if err != nil {
		return nil, err
	}

	c.httpClient.Transport = &apiTransport{authToken: authToken}

	return c, nil
}
