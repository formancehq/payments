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
	GetWallets(ctx context.Context, cursor string, pageSize int) (*WalletsResponse, error)
	GetBalances(ctx context.Context, cursor string, pageSize int) (*BalancesResponse, error)
	GetTransactions(ctx context.Context, cursor string, pageSize int) (*TransactionsResponse, error)
}

const defaultBaseURL = "https://api.prime.coinbase.com"

type client struct {
	httpClient  httpwrapper.Client
	baseURL     string
	apiKey      string
	apiSecret   string
	passphrase  string
	portfolioID string
}

func New(connectorName, apiKey, apiSecret, passphrase, portfolioID string) Client {
	return NewWithBaseURL(connectorName, apiKey, apiSecret, passphrase, portfolioID, defaultBaseURL)
}

func NewWithBaseURL(connectorName, apiKey, apiSecret, passphrase, portfolioID, baseURL string) Client {
	c := &client{
		baseURL:     baseURL,
		apiKey:      apiKey,
		apiSecret:   apiSecret,
		passphrase:  passphrase,
		portfolioID: portfolioID,
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

	req.Header.Set("X-CB-ACCESS-KEY", c.apiKey)
	req.Header.Set("X-CB-ACCESS-SIGNATURE", signature)
	req.Header.Set("X-CB-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("X-CB-ACCESS-PASSPHRASE", c.passphrase)
	req.Header.Set("Content-Type", "application/json")

	return nil
}

func (c *client) GetWallets(ctx context.Context, cursor string, pageSize int) (*WalletsResponse, error) {
	endpoint := fmt.Sprintf("%s/v1/portfolios/%s/wallets?limit=%d&sort_direction=ASC", c.baseURL, c.portfolioID, pageSize)
	if cursor != "" {
		endpoint += fmt.Sprintf("&cursor=%s", cursor)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.signRequest(req, ""); err != nil {
		return nil, err
	}

	var response WalletsResponse
	var errorResponse ErrorResponse
	statusCode, err := c.httpClient.Do(ctx, req, &response, &errorResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallets (status %d): %w", statusCode, err)
	}

	return &response, nil
}

func (c *client) GetBalances(ctx context.Context, cursor string, pageSize int) (*BalancesResponse, error) {
	endpoint := fmt.Sprintf("%s/v1/portfolios/%s/balances?limit=%d&sort_direction=ASC", c.baseURL, c.portfolioID, pageSize)
	if cursor != "" {
		endpoint += fmt.Sprintf("&cursor=%s", cursor)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.signRequest(req, ""); err != nil {
		return nil, err
	}

	var response BalancesResponse
	var errorResponse ErrorResponse
	statusCode, err := c.httpClient.Do(ctx, req, &response, &errorResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to get balances (status %d): %w", statusCode, err)
	}

	return &response, nil
}

func (c *client) GetTransactions(ctx context.Context, cursor string, pageSize int) (*TransactionsResponse, error) {
	endpoint := fmt.Sprintf("%s/v1/portfolios/%s/transactions?limit=%d&sort_direction=ASC", c.baseURL, c.portfolioID, pageSize)
	if cursor != "" {
		endpoint += fmt.Sprintf("&cursor=%s", cursor)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.signRequest(req, ""); err != nil {
		return nil, err
	}

	var response TransactionsResponse
	var errorResponse ErrorResponse
	statusCode, err := c.httpClient.Do(ctx, req, &response, &errorResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions (status %d): %w", statusCode, err)
	}

	return &response, nil
}

// Wallet represents a Coinbase Prime wallet.
type Wallet struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Symbol    string    `json:"symbol"`
	Type      string    `json:"type"` // TRADING, VAULT, ONCHAIN
	CreatedAt time.Time `json:"created_at"`
}

// Balance represents a Coinbase Prime portfolio balance.
type Balance struct {
	Symbol             string `json:"symbol"`
	Amount             string `json:"amount"`
	Holds              string `json:"holds"`
	WithdrawableAmount string `json:"withdrawable_amount"`
	FiatAmount         string `json:"fiat_amount"`
}

// Transaction represents a Coinbase Prime transaction.
type Transaction struct {
	ID            string     `json:"id"`
	WalletID      string     `json:"wallet_id"`
	PortfolioID   string     `json:"portfolio_id"`
	Type          string     `json:"type"`   // DEPOSIT, WITHDRAWAL, INTERNAL_TRANSFER, ...
	Status        string     `json:"status"` // TRANSACTION_PENDING, TRANSACTION_COMPLETED, TRANSACTION_FAILED
	Symbol        string     `json:"symbol"`
	Amount        string     `json:"amount"`
	Fees          string     `json:"fees"`
	FeeSymbol     string     `json:"fee_symbol"`
	CreatedAt     time.Time  `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	TransferFrom  string     `json:"transfer_from"`
	TransferTo    string     `json:"transfer_to"`
	NetworkFees   string     `json:"network_fees"`
	Network       string     `json:"network"`
	BlockchainIDs []string   `json:"blockchain_ids"`
}

// Pagination represents cursor-based pagination from Coinbase Prime.
type Pagination struct {
	NextCursor    string `json:"next_cursor"`
	SortDirection string `json:"sort_direction"`
	HasNext       bool   `json:"has_next"`
}

// WalletsResponse wraps wallets with pagination.
type WalletsResponse struct {
	Wallets    []Wallet   `json:"wallets"`
	Pagination Pagination `json:"pagination"`
}

// BalancesResponse wraps balances with pagination.
type BalancesResponse struct {
	Balances   []Balance  `json:"balances"`
	Pagination Pagination `json:"pagination"`
}

// TransactionsResponse wraps transactions with pagination.
type TransactionsResponse struct {
	Transactions []Transaction `json:"transactions"`
	Pagination   Pagination    `json:"pagination"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Message string `json:"message"`
}
