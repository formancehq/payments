package client

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetAccounts(ctx context.Context) ([]Account, error)
	GetTransfers(ctx context.Context, cursor string, pageSize int) (*TransfersResponse, error)
}

const defaultBaseURL = "https://api.exchange.coinbase.com"

type client struct {
	httpClient httpwrapper.Client
	baseURL    string
	apiKey     string
	apiSecret  string
	passphrase string
}

func New(connectorName, apiKey, apiSecret, passphrase string) Client {
	return NewWithBaseURL(connectorName, apiKey, apiSecret, passphrase, defaultBaseURL)
}

func NewWithBaseURL(connectorName, apiKey, apiSecret, passphrase, baseURL string) Client {
	c := &client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		passphrase: passphrase,
	}

	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{
			Transport: http.DefaultTransport,
		}),
	}
	c.httpClient = httpwrapper.NewClient(config)

	return c
}

func (c *client) signRequest(req *http.Request, body string) error {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	message := timestamp + req.Method + req.URL.Path + body
	if req.URL.RawQuery != "" {
		message = timestamp + req.Method + req.URL.Path + "?" + req.URL.RawQuery + body
	}

	secret, err := base64.StdEncoding.DecodeString(c.apiSecret)
	if err != nil {
		return fmt.Errorf("failed to decode API secret: %w", err)
	}

	h := hmac.New(sha256.New, secret)
	h.Write([]byte(message))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	req.Header.Set("CB-ACCESS-KEY", c.apiKey)
	req.Header.Set("CB-ACCESS-SIGN", signature)
	req.Header.Set("CB-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("CB-ACCESS-PASSPHRASE", c.passphrase)
	req.Header.Set("Content-Type", "application/json")

	return nil
}

func (c *client) GetAccounts(ctx context.Context) ([]Account, error) {
	endpoint := fmt.Sprintf("%s/accounts", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.signRequest(req, ""); err != nil {
		return nil, err
	}

	var accounts []Account
	var errorResponse ErrorResponse
	statusCode, err := c.httpClient.Do(ctx, req, &accounts, &errorResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts (status %d): %w", statusCode, err)
	}

	return accounts, nil
}

func (c *client) GetTransfers(ctx context.Context, cursor string, pageSize int) (*TransfersResponse, error) {
	endpoint := fmt.Sprintf("%s/transfers?limit=%d", c.baseURL, pageSize)
	if cursor != "" {
		endpoint += fmt.Sprintf("&after=%s", cursor)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.signRequest(req, ""); err != nil {
		return nil, err
	}

	var transfers []Transfer
	var errorResponse ErrorResponse
	statusCode, err := c.httpClient.Do(ctx, req, &transfers, &errorResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to get transfers (status %d): %w", statusCode, err)
	}

	var nextCursor string
	if len(transfers) > 0 {
		nextCursor = transfers[len(transfers)-1].ID
	}

	return &TransfersResponse{
		Transfers:  transfers,
		NextCursor: nextCursor,
		HasMore:    len(transfers) == pageSize,
	}, nil
}

// Account represents a Coinbase Exchange account (wallet).
type Account struct {
	ID             string `json:"id"`
	Currency       string `json:"currency"`
	Balance        string `json:"balance"`
	Available      string `json:"available"`
	Hold           string `json:"hold"`
	ProfileID      string `json:"profile_id"`
	TradingEnabled bool   `json:"trading_enabled"`
}

// Transfer represents a deposit or withdrawal.
type Transfer struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"` // deposit, withdraw, internal_deposit, internal_withdraw
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CanceledAt  *time.Time `json:"canceled_at"`
	ProcessedAt *time.Time `json:"processed_at"`
	Amount      string     `json:"amount"`
	Currency    string     `json:"currency"`
	UserNonce   *string    `json:"user_nonce"`
	Details     TransferDetails `json:"details"`
}

// TransferDetails contains additional transfer metadata.
type TransferDetails struct {
	CoinbaseAccountID       string `json:"coinbase_account_id,omitempty"`
	CoinbaseTransactionID   string `json:"coinbase_transaction_id,omitempty"`
	CoinbasePaymentMethodID string `json:"coinbase_payment_method_id,omitempty"`
	DestinationTag          string `json:"destination_tag,omitempty"`
	CryptoAddress           string `json:"crypto_address,omitempty"`
	CryptoTransactionHash   string `json:"crypto_transaction_hash,omitempty"`
	SentToAddress           string `json:"sent_to_address,omitempty"`
}

// TransfersResponse wraps transfers with pagination info.
type TransfersResponse struct {
	Transfers  []Transfer
	NextCursor string
	HasMore    bool
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Message string `json:"message"`
}
