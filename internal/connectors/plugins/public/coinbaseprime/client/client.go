package client

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetPortfolios(ctx context.Context) (*PortfoliosResponse, error)
	GetWallets(ctx context.Context, cursor string, pageSize int) (*WalletsResponse, error)
	GetBalances(ctx context.Context, cursor string, pageSize int) (*BalancesResponse, error)
	GetBalancesForSymbol(ctx context.Context, symbol string, cursor string, pageSize int) (*BalancesResponse, error)
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
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}),
	}
	c.httpClient = httpwrapper.NewClient(config)

	return c
}

func (c *client) signRequest(req *http.Request, body string) error {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Coinbase Prime signature must include only the path (no query params).
	// See: https://docs.cdp.coinbase.com/prime/rest-api/requests
	message := timestamp + req.Method + req.URL.Path + body

	// Coinbase Prime expects the signing key to be used as raw string bytes
	// for the HMAC computation, NOT base64-decoded.
	h := hmac.New(sha256.New, []byte(c.apiSecret))
	h.Write([]byte(message))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	req.Header.Set("X-CB-ACCESS-KEY", c.apiKey)
	req.Header.Set("X-CB-ACCESS-SIGNATURE", signature)
	req.Header.Set("X-CB-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("X-CB-ACCESS-PASSPHRASE", c.passphrase)
	req.Header.Set("Content-Type", "application/json")

	return nil
}

func (c *client) GetPortfolios(ctx context.Context) (*PortfoliosResponse, error) {
	endpoint := fmt.Sprintf("%s/v1/portfolios", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.signRequest(req, ""); err != nil {
		return nil, err
	}

	var response PortfoliosResponse
	var errorResponse ErrorResponse
	statusCode, err := c.httpClient.Do(ctx, req, &response, &errorResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolios (status %d, message: %s): %w", statusCode, errorResponse.Message, err)
	}

	return &response, nil
}

func (c *client) GetWallets(ctx context.Context, cursor string, pageSize int) (*WalletsResponse, error) {
	endpoint, err := c.buildPortfolioEndpoint("wallets", cursor, pageSize)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("failed to get wallets (status %d, message: %s): %w", statusCode, errorResponse.Message, err)
	}

	return &response, nil
}

func (c *client) GetBalances(ctx context.Context, cursor string, pageSize int) (*BalancesResponse, error) {
	endpoint, err := c.buildPortfolioEndpoint("balances", cursor, pageSize)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("failed to get balances (status %d, message: %s): %w", statusCode, errorResponse.Message, err)
	}

	return &response, nil
}

func (c *client) GetBalancesForSymbol(ctx context.Context, symbol string, cursor string, pageSize int) (*BalancesResponse, error) {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("missing symbol for balances filtering")
	}

	var allFiltered []Balance
	currentCursor := cursor

	for {
		response, err := c.GetBalances(ctx, currentCursor, pageSize)
		if err != nil {
			return nil, err
		}

		for _, balance := range response.Balances {
			if strings.EqualFold(balance.Symbol, symbol) {
				allFiltered = append(allFiltered, balance)
			}
		}

		if !response.Pagination.HasNext {
			return &BalancesResponse{
				Balances:   allFiltered,
				Pagination: response.Pagination,
			}, nil
		}
		currentCursor = response.Pagination.NextCursor
	}
}

func (c *client) GetTransactions(ctx context.Context, cursor string, pageSize int) (*TransactionsResponse, error) {
	endpoint, err := c.buildPortfolioEndpoint("transactions", cursor, pageSize)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("failed to get transactions (status %d, message: %s): %w", statusCode, errorResponse.Message, err)
	}

	return &response, nil
}

func (c *client) buildPortfolioEndpoint(resource, cursor string, pageSize int) (string, error) {
	endpoint, err := url.Parse(fmt.Sprintf("%s/v1/portfolios/%s/%s", c.baseURL, c.portfolioID, resource))
	if err != nil {
		return "", fmt.Errorf("failed to parse endpoint: %w", err)
	}

	query := endpoint.Query()
	query.Set("limit", strconv.Itoa(pageSize))
	query.Set("sort_direction", "ASC")
	if cursor != "" {
		query.Set("cursor", cursor)
	}

	endpoint.RawQuery = query.Encode()
	return endpoint.String(), nil
}

// Portfolio represents a Coinbase Prime portfolio.
type Portfolio struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	EntityID       string    `json:"entity_id"`
	OrganizationID string    `json:"organization_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// PortfoliosResponse wraps portfolios from the list endpoint.
type PortfoliosResponse struct {
	Portfolios []Portfolio `json:"portfolios"`
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

// TransferEndpoint represents a source or destination in a Coinbase Prime transaction.
type TransferEndpoint struct {
	Type              string `json:"type"`  // WALLET, PAYMENT_METHOD, ...
	Value             string `json:"value"` // wallet ID or payment method ID
	Address           string `json:"address"`
	AccountIdentifier string `json:"account_identifier"`
}

// Transaction represents a Coinbase Prime transaction.
type Transaction struct {
	ID            string            `json:"id"`
	WalletID      string            `json:"wallet_id"`
	PortfolioID   string            `json:"portfolio_id"`
	Type          string            `json:"type"`   // DEPOSIT, WITHDRAWAL, INTERNAL_TRANSFER, CONVERSION, ...
	Status        string            `json:"status"` // TRANSACTION_PENDING, TRANSACTION_DONE, TRANSACTION_IMPORT_PENDING, ...
	Symbol        string            `json:"symbol"`
	Amount        string            `json:"amount"`
	Fees          string            `json:"fees"`
	FeeSymbol     string            `json:"fee_symbol"`
	CreatedAt     time.Time         `json:"created_at"`
	CompletedAt   *time.Time        `json:"completed_at"`
	TransferFrom  *TransferEndpoint `json:"transfer_from"`
	TransferTo    *TransferEndpoint `json:"transfer_to"`
	NetworkFees   string            `json:"network_fees"`
	Network       string            `json:"network"`
	BlockchainIDs []string          `json:"blockchain_ids"`
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
