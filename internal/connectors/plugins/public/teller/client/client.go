package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
)

const (
	sandboxBaseURL    = "https://api.teller.io"
	productionBaseURL = "https://api.teller.io"
)

// Teller API model types

type Account struct {
	ID            string          `json:"id"`
	Currency      string          `json:"currency"`
	EnrollmentID  string          `json:"enrollment_id"`
	Institution   Institution     `json:"institution"`
	LastFour      string          `json:"last_four"`
	Links         json.RawMessage `json:"links"`
	Name          string          `json:"name"`
	Status        string          `json:"status"`
	Subtype       string          `json:"subtype"`
	Type          string          `json:"type"`
}

type Institution struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Balance struct {
	AccountID string `json:"account_id"`
	Available string `json:"available"`
	Ledger    string `json:"ledger"`
	Links     json.RawMessage `json:"links"`
}

type Transaction struct {
	ID          string          `json:"id"`
	AccountID   string          `json:"account_id"`
	Amount      string          `json:"amount"`
	Date        string          `json:"date"`
	Description string          `json:"description"`
	Details     TransactionDetails `json:"details"`
	Links       json.RawMessage `json:"links"`
	RunningBalance *string      `json:"running_balance"`
	Status      string          `json:"status"`
	Type        string          `json:"type"`
}

type TransactionDetails struct {
	Category      string `json:"category"`
	Counterparty  Counterparty `json:"counterparty"`
	ProcessingStatus string `json:"processing_status"`
}

type Counterparty struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	ListAccounts(ctx context.Context, accessToken string) ([]Account, error)
	GetBalance(ctx context.Context, accessToken string, accountID string) (*Balance, error)
	ListTransactions(ctx context.Context, accessToken string, accountID string, fromID string, count int) ([]Transaction, error)
}

type client struct {
	httpClient *http.Client
	baseURL    string
}

func New(connectorName string, isSandbox bool) Client {
	baseURL := productionBaseURL
	if isSandbox {
		baseURL = sandboxBaseURL
	}

	return &client{
		httpClient: metrics.NewHTTPClient(connectorName, models.DefaultConnectorClientTimeout),
		baseURL:    baseURL,
	}
}

func (c *client) ListAccounts(ctx context.Context, accessToken string) ([]Account, error) {
	ctx = metrics.OperationContext(ctx, "list_accounts")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/accounts", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(accessToken, "")

	var accounts []Account
	if err := c.do(req, &accounts); err != nil {
		return nil, err
	}
	return accounts, nil
}

func (c *client) GetBalance(ctx context.Context, accessToken string, accountID string) (*Balance, error) {
	ctx = metrics.OperationContext(ctx, "get_balance")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/accounts/%s/balances", c.baseURL, accountID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(accessToken, "")

	var balance Balance
	if err := c.do(req, &balance); err != nil {
		return nil, err
	}
	return &balance, nil
}

func (c *client) ListTransactions(ctx context.Context, accessToken string, accountID string, fromID string, count int) ([]Transaction, error) {
	ctx = metrics.OperationContext(ctx, "list_transactions")

	url := fmt.Sprintf("%s/accounts/%s/transactions", c.baseURL, accountID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(accessToken, "")

	q := req.URL.Query()
	if fromID != "" {
		q.Set("from_id", fromID)
	}
	if count > 0 {
		q.Set("count", fmt.Sprintf("%d", count))
	}
	req.URL.RawQuery = q.Encode()

	var transactions []Transaction
	if err := c.do(req, &transactions); err != nil {
		return nil, err
	}
	return transactions, nil
}

func (c *client) do(req *http.Request, v any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")
		return fmt.Errorf("rate limited by teller (retry-after: %s)", retryAfter)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		var tellerErr struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if decErr := json.NewDecoder(resp.Body).Decode(&tellerErr); decErr == nil && tellerErr.Error.Code != "" {
			return fmt.Errorf("teller API error (status %d): %s: %s", resp.StatusCode, tellerErr.Error.Code, tellerErr.Error.Message)
		}
		return fmt.Errorf("teller API error: unexpected status code %d", resp.StatusCode)
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

