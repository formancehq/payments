package client

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client

type Client interface {
	// Vault operations
	GetVaultAccounts(ctx context.Context, params GetVaultAccountsParams) (*VaultAccountsPagedResponse, error)
	GetVaultAccountAsset(ctx context.Context, vaultAccountID, assetID string) (*VaultAsset, error)

	// External wallet operations
	GetExternalWallets(ctx context.Context) ([]ExternalWallet, error)
	GetInternalWallets(ctx context.Context) ([]InternalWallet, error)

	// Transaction operations
	GetTransactions(ctx context.Context, params GetTransactionsParams) ([]TransactionResponse, error)
	GetTransaction(ctx context.Context, txID string) (*TransactionResponse, error)
	CreateTransaction(ctx context.Context, req CreateTransactionRequest) (*CreateTransactionResponse, error)

	// Asset operations
	GetSupportedAssets(ctx context.Context) ([]AssetTypeResponse, error)
}

type client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	privateKey *rsa.PrivateKey
}

func New(
	apiKey string,
	privateKeyPEM string,
	baseURL string,
) (Client, error) {
	privateKey, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		apiKey:     apiKey,
		privateKey: privateKey,
	}, nil
}

func parsePrivateKey(pemString string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemString))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS1 format
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not an RSA key")
	}

	return rsaKey, nil
}

func (c *client) signRequest(path string, body []byte) (string, error) {
	now := time.Now()
	nonce := now.UnixNano()

	// Create body hash
	var bodyHash string
	if len(body) > 0 {
		hash := sha256.Sum256(body)
		bodyHash = hex.EncodeToString(hash[:])
	} else {
		hash := sha256.Sum256([]byte{})
		bodyHash = hex.EncodeToString(hash[:])
	}

	claims := jwt.MapClaims{
		"uri":      path,
		"nonce":    nonce,
		"iat":      now.Unix(),
		"exp":      now.Add(30 * time.Second).Unix(),
		"sub":      c.apiKey,
		"bodyHash": bodyHash,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(c.privateKey)
}

func (c *client) doRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	fullURL := c.baseURL + path

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = strings.NewReader(string(body))
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	token, err := c.signRequest(path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetVaultAccountsParams contains parameters for listing vault accounts
type GetVaultAccountsParams struct {
	NamePrefix       string
	NameSuffix       string
	MinAmountThreshold float64
	AssetID          string
	OrderBy          string
	Limit            int
	Before           string
	After            string
}

// GetVaultAccounts retrieves a paginated list of vault accounts
func (c *client) GetVaultAccounts(ctx context.Context, params GetVaultAccountsParams) (*VaultAccountsPagedResponse, error) {
	path := "/vault/accounts_paged"

	// Build query parameters
	queryParts := []string{}
	if params.NamePrefix != "" {
		queryParts = append(queryParts, fmt.Sprintf("namePrefix=%s", params.NamePrefix))
	}
	if params.Limit > 0 {
		queryParts = append(queryParts, fmt.Sprintf("limit=%d", params.Limit))
	}
	if params.Before != "" {
		queryParts = append(queryParts, fmt.Sprintf("before=%s", params.Before))
	}
	if params.After != "" {
		queryParts = append(queryParts, fmt.Sprintf("after=%s", params.After))
	}

	if len(queryParts) > 0 {
		path += "?" + strings.Join(queryParts, "&")
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result VaultAccountsPagedResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// GetVaultAccountAsset retrieves a specific asset within a vault account
func (c *client) GetVaultAccountAsset(ctx context.Context, vaultAccountID, assetID string) (*VaultAsset, error) {
	path := fmt.Sprintf("/vault/accounts/%s/%s", vaultAccountID, assetID)

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result VaultAsset
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// GetExternalWallets retrieves all external wallets
func (c *client) GetExternalWallets(ctx context.Context) ([]ExternalWallet, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/external_wallets", nil)
	if err != nil {
		return nil, err
	}

	var result []ExternalWallet
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// GetInternalWallets retrieves all internal wallets
func (c *client) GetInternalWallets(ctx context.Context) ([]InternalWallet, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/internal_wallets", nil)
	if err != nil {
		return nil, err
	}

	var result []InternalWallet
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// GetTransactionsParams contains parameters for listing transactions
type GetTransactionsParams struct {
	Before      string
	After       string
	Status      string
	OrderBy     string
	Limit       int
	SourceType  string
	SourceID    string
	DestType    string
	DestID      string
	Assets      string
	TxHash      string
}

// GetTransactions retrieves a list of transactions
func (c *client) GetTransactions(ctx context.Context, params GetTransactionsParams) ([]TransactionResponse, error) {
	path := "/transactions"

	queryParts := []string{}
	if params.Status != "" {
		queryParts = append(queryParts, fmt.Sprintf("status=%s", params.Status))
	}
	if params.Limit > 0 {
		queryParts = append(queryParts, fmt.Sprintf("limit=%d", params.Limit))
	}
	if params.Before != "" {
		queryParts = append(queryParts, fmt.Sprintf("before=%s", params.Before))
	}
	if params.After != "" {
		queryParts = append(queryParts, fmt.Sprintf("after=%s", params.After))
	}
	if params.OrderBy != "" {
		queryParts = append(queryParts, fmt.Sprintf("orderBy=%s", params.OrderBy))
	}

	if len(queryParts) > 0 {
		path += "?" + strings.Join(queryParts, "&")
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result []TransactionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// GetTransaction retrieves a specific transaction by ID
func (c *client) GetTransaction(ctx context.Context, txID string) (*TransactionResponse, error) {
	path := fmt.Sprintf("/transactions/%s", txID)

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result TransactionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// CreateTransactionRequest contains the request body for creating a transaction
type CreateTransactionRequest struct {
	AssetID             string                 `json:"assetId"`
	Source              TransferPeerPath       `json:"source"`
	Destination         DestinationTransferPeerPath `json:"destination"`
	Amount              string                 `json:"amount"`
	Fee                 string                 `json:"fee,omitempty"`
	FeeLevel            string                 `json:"feeLevel,omitempty"`
	Note                string                 `json:"note,omitempty"`
	ExternalTxID        string                 `json:"externalTxId,omitempty"`
	TreatAsGrossAmount  bool                   `json:"treatAsGrossAmount,omitempty"`
	Operation           string                 `json:"operation,omitempty"`
}

// TransferPeerPath represents the source or destination of a transfer
type TransferPeerPath struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
}

// DestinationTransferPeerPath represents the destination with optional one-time address
type DestinationTransferPeerPath struct {
	Type           string                  `json:"type"`
	ID             string                  `json:"id,omitempty"`
	OneTimeAddress *OneTimeAddress         `json:"oneTimeAddress,omitempty"`
}

// OneTimeAddress represents a one-time address destination
type OneTimeAddress struct {
	Address string `json:"address"`
	Tag     string `json:"tag,omitempty"`
}

// CreateTransactionResponse contains the response from creating a transaction
type CreateTransactionResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// CreateTransaction creates a new transaction
func (c *client) CreateTransaction(ctx context.Context, req CreateTransactionRequest) (*CreateTransactionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, "/transactions", body)
	if err != nil {
		return nil, err
	}

	var result CreateTransactionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// GetSupportedAssets retrieves all supported assets
func (c *client) GetSupportedAssets(ctx context.Context) ([]AssetTypeResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/supported_assets", nil)
	if err != nil {
		return nil, err
	}

	var result []AssetTypeResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}
