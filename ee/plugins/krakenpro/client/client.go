package client

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
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
	GetBalance(ctx context.Context) (*BalanceResponse, error)
	GetLedgers(ctx context.Context, offset int, startTime int64) (*LedgersResponse, error)
}

const defaultBaseURL = "https://api.kraken.com"

type client struct {
	httpClient httpwrapper.Client
	baseURL    string
	apiKey     string
	privateKey []byte // base64-decoded private key
}

func New(connectorName, apiKey, privateKeyB64 string) (Client, error) {
	return NewWithBaseURL(connectorName, apiKey, privateKeyB64, defaultBaseURL)
}

func NewWithBaseURL(connectorName, apiKey, privateKeyB64, baseURL string) (Client, error) {
	decodedKey, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	c := &client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		privateKey: decodedKey,
	}

	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}),
	}
	c.httpClient = httpwrapper.NewClient(config)

	return c, nil
}

// signRequest signs a Kraken API request using HMAC-SHA512.
// Signature = Base64(HMAC-SHA512(path + SHA256(nonce + postdata), privateKey))
// where SHA256 produces raw bytes (not hex) and path is byte-concatenated.
func (c *client) signRequest(uriPath string, nonce int64, postData string) string {
	// Step 1: SHA256(nonce_string + postdata) → raw bytes
	nonceStr := strconv.FormatInt(nonce, 10)
	sha256Hash := sha256.Sum256([]byte(nonceStr + postData))

	// Step 2: binary concatenation: []byte(path) + sha256_bytes
	message := make([]byte, 0, len(uriPath)+sha256.Size)
	message = append(message, uriPath...)
	message = append(message, sha256Hash[:]...)

	// Step 3: HMAC-SHA512 with base64-decoded private key
	mac := hmac.New(sha512.New, c.privateKey)
	mac.Write(message)

	// Step 4: Base64 encode the result
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (c *client) doPrivateRequest(ctx context.Context, uriPath string, params url.Values, result any) error {
	nonce := time.Now().UnixMicro()
	params.Set("nonce", strconv.FormatInt(nonce, 10))

	postData := params.Encode()
	signature := c.signRequest(uriPath, nonce, postData)

	endpoint := c.baseURL + uriPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(postData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("API-Key", c.apiKey)
	req.Header.Set("API-Sign", signature)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var errResp ErrorResponse
	statusCode, err := c.httpClient.Do(ctx, req, result, &errResp)
	if err != nil {
		errDetail := ""
		if len(errResp.Error) > 0 {
			errDetail = " [" + strings.Join(errResp.Error, "; ") + "]"
		}
		return fmt.Errorf("request to %s failed (status %d)%s: %w", uriPath, statusCode, errDetail, err)
	}

	return nil
}

func (c *client) GetBalance(ctx context.Context) (*BalanceResponse, error) {
	params := url.Values{}
	var response BalanceResponse
	if err := c.doPrivateRequest(ctx, "/0/private/Balance", params, &response); err != nil {
		return nil, err
	}

	if err := response.CheckErrors(); err != nil {
		return nil, fmt.Errorf("get balance: %w", err)
	}

	return &response, nil
}

func (c *client) GetLedgers(ctx context.Context, offset int, startTime int64) (*LedgersResponse, error) {
	params := url.Values{}
	if offset > 0 {
		params.Set("ofs", strconv.Itoa(offset))
	}
	if startTime > 0 {
		params.Set("start", strconv.FormatInt(startTime, 10))
	}

	var response LedgersResponse
	if err := c.doPrivateRequest(ctx, "/0/private/Ledgers", params, &response); err != nil {
		return nil, err
	}

	if err := response.CheckErrors(); err != nil {
		return nil, fmt.Errorf("get ledgers: %w", err)
	}

	return &response, nil
}

// KrakenError represents an API error from Kraken.
type KrakenError struct {
	Errors []string
}

func (e *KrakenError) Error() string {
	return strings.Join(e.Errors, "; ")
}

func (e *KrakenError) IsRateLimited() bool {
	for _, err := range e.Errors {
		if strings.HasPrefix(err, "EAPI:Rate limit exceeded") {
			return true
		}
	}
	return false
}

// BalanceResponse is the response from POST /0/private/Balance.
type BalanceResponse struct {
	Error  []string          `json:"error"`
	Result map[string]string `json:"result"`
}

func (r *BalanceResponse) CheckErrors() error {
	if len(r.Error) > 0 {
		return &KrakenError{Errors: r.Error}
	}
	return nil
}

// LedgerEntry represents a single ledger entry from Kraken.
type LedgerEntry struct {
	RefID   string  `json:"refid"`
	Time    float64 `json:"time"`
	Type    string  `json:"type"`
	Subtype string  `json:"subtype"`
	Aclass  string  `json:"aclass"`
	Asset   string  `json:"asset"`
	Amount  string  `json:"amount"`
	Fee     string  `json:"fee"`
	Balance string  `json:"balance"`
}

// LedgersResult contains the ledgers map and count.
type LedgersResult struct {
	Ledgers map[string]LedgerEntry `json:"ledger"`
	Count   int                    `json:"count"`
}

// LedgersResponse is the response from POST /0/private/Ledgers.
type LedgersResponse struct {
	Error  []string      `json:"error"`
	Result LedgersResult `json:"result"`
}

func (r *LedgersResponse) CheckErrors() error {
	if len(r.Error) > 0 {
		return &KrakenError{Errors: r.Error}
	}
	return nil
}

// ErrorResponse is used by httpwrapper for non-2xx responses.
type ErrorResponse struct {
	Error []string `json:"error"`
}
