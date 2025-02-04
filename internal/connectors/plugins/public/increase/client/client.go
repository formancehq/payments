package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/httpwrapper"
	"github.com/formancehq/payments/pkg/metrics"
)

type Client interface {
	GetAccounts(ctx context.Context, lastID string, pageSize int64) ([]*Account, string, bool, error)
	GetAccountBalances(ctx context.Context, accountID string) ([]*Balance, error)
	GetTransactions(ctx context.Context, lastID string, pageSize int64) ([]*Transaction, string, bool, error)
	GetPendingTransactions(ctx context.Context, lastID string, pageSize int64) ([]*Transaction, string, bool, error)
	GetDeclinedTransactions(ctx context.Context, lastID string, pageSize int64) ([]*Transaction, string, bool, error)
	GetExternalAccounts(ctx context.Context, lastID string, pageSize int64) ([]*ExternalAccount, string, bool, error)
	CreateExternalAccount(ctx context.Context, req *CreateExternalAccountRequest) (*ExternalAccount, error)
	CreateTransfer(ctx context.Context, req *CreateTransferRequest) (*Transfer, error)
	CreateACHTransfer(ctx context.Context, req *CreateACHTransferRequest) (*Transfer, error)
	CreateWireTransfer(ctx context.Context, req *CreateWireTransferRequest) (*Transfer, error)
	CreateCheckTransfer(ctx context.Context, req *CreateCheckTransferRequest) (*Transfer, error)
	CreateRTPTransfer(ctx context.Context, req *CreateRTPTransferRequest) (*Transfer, error)
}

type client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

func NewClient(apiKey string) Client {
	return &client{
		httpClient: &http.Client{
			Transport: metrics.NewTransport("increase", metrics.TransportOpts{}),
		},
		baseURL: "https://api.increase.com",
		apiKey:  apiKey,
	}
}

func (c *client) newRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}
	u.Path = path.Join(u.Path, endpoint)

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (c *client) do(req *http.Request, v interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
