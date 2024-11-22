package testserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
)

type Client struct {
	baseUrl        string
	internalClient httpwrapper.Client
	transport      http.RoundTripper
}

func NewClient(urlStr string, transport http.RoundTripper) (*Client, error) {
	config := &httpwrapper.Config{}
	internalClient := httpwrapper.NewClient(config)
	return &Client{
		baseUrl:        urlStr,
		transport:      transport,
		internalClient: internalClient,
	}, nil
}

func (c *Client) wrapError(err error, method, path string, status int, errBody map[string]interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("error with status %d for %s to '%s': %w, body: %+v", status, method, path, err, errBody)
}

func (c *Client) Get(ctx context.Context, path string, resBody any) error {
	method := http.MethodGet
	req, err := http.NewRequestWithContext(ctx, method, c.baseUrl+path, nil)
	if err != nil {
		return err
	}

	var errBody map[string]interface{}
	status, err := c.internalClient.Do(ctx, req, resBody, &errBody)
	return c.wrapError(err, method, path, status, errBody)
}

func (c *Client) Do(ctx context.Context, method string, path string, body any, resBody any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseUrl+path, bytes.NewReader(b))
	if err != nil {
		return err
	}

	var errBody map[string]interface{}
	status, err := c.internalClient.Do(ctx, req, resBody, &errBody)
	return c.wrapError(err, method, path, status, errBody)
}
