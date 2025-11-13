package client

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ----- Public API -----

// Client provides authenticated access to the Bitstamp API v2.
// It supports multiple accounts, each with its own API credentials,
// and implements Bitstamp's HMAC-SHA256 signature authentication.
type Client interface {
	GetTransactions(ctx context.Context, params TransactionsParams) ([]Transaction, error)
	GetTransactionsForAccount(ctx context.Context, account *Account, params TransactionsParams) ([]Transaction, error)
	GetAccounts(ctx context.Context, page, pageSize int) ([]*Account, error)
	GetAccountBalances(ctx context.Context, account *Account) ([]*Balance, error)
	GetAllAccounts() []*Account
}

type client struct {
	accounts []*Account

	baseURL    *url.URL
	httpClient *http.Client
	underlying http.RoundTripper
	timeout    time.Duration
}

// New creates a Bitstamp client with the provided options.
// The client supports multiple accounts, each authenticated independently.
func New(opts ...Option) Client {
	c := &client{
		baseURL: mustParseURL("https://www.bitstamp.net"),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Base httpClient is set up without auth; credentials are per-account
	if c.httpClient == nil {
		c.httpClient = &http.Client{
			Transport: c.underlying, // may be nil → http.DefaultTransport
			Timeout:   c.timeoutOrDefault(),
		}
	}

	return c
}

// Options for New
type Option func(*client)

func WithBaseURL(raw string) Option {
	return func(c *client) { c.baseURL = mustParseURL(raw) }
}

func WithHTTPClient(hc *http.Client) Option {
	return func(c *client) { c.httpClient = hc }
}

func WithUnderlyingTransport(rt http.RoundTripper) Option {
	return func(c *client) { c.underlying = rt }
}

func WithTimeout(d time.Duration) Option {
	return func(c *client) { c.timeout = d }
}

func WithAccounts(accounts []*Account) Option {
	return func(c *client) { c.accounts = accounts }
}

// ----- Implementation -----

func (c *client) timeoutOrDefault() time.Duration {
	if c.timeout > 0 {
		return c.timeout
	}
	return 30 * time.Second
}

// getMainAccount returns the account with name "main"
func (c *client) getMainAccount() (*Account, error) {
	for _, account := range c.accounts {
		if account.Name == "main" {
			return account, nil
		}
	}
	return nil, fmt.Errorf("bitstamp: no account with name 'main' found")
}

// httpClientForAccount creates an authenticated HTTP client for a specific account
func (c *client) httpClientForAccount(account *Account) *http.Client {
	tr := &signingTransport{
		APIKey:     account.APIKey,
		APISecret:  []byte(account.ApiSecret),
		Underlying: c.underlying,
	}

	if c.httpClient == nil {
		return &http.Client{
			Transport: tr,
			Timeout:   c.timeoutOrDefault(),
		}
	}

	// Clone the base client and wrap with account-specific auth
	clone := *c.httpClient
	clone.Transport = tr
	return &clone
}

// GetAllAccounts returns all configured accounts
func (c *client) GetAllAccounts() []*Account {
	return c.accounts
}

// ----- Signing Transport (Bitstamp v2) -----

// signingTransport wraps an HTTP transport to automatically sign requests
// using Bitstamp's v2 authentication (HMAC-SHA256 signature).
type signingTransport struct {
	APIKey     string
	APISecret  []byte
	Underlying http.RoundTripper
}

// RoundTrip implements http.RoundTripper by adding Bitstamp v2 authentication headers.
// The signature is computed over: BITSTAMP + API_KEY + method + host + path + query +
// content-type + nonce + timestamp + version + body
func (t *signingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt := t.Underlying
	if rt == nil {
		rt = http.DefaultTransport
	}

	// Timestamp (milliseconds) and nonce per request
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano()/1e6)
	nonce := newNonce()

	// Buffer the request body for signing, then restore it.
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		_ = req.Body.Close()
	}
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	// Always set ContentLength explicitly (even for nil/empty body)
	req.ContentLength = int64(len(bodyBytes))

	// Content-Type used in the signature. If request body is empty, don't add Content-Type.
	var contentType string
	if len(bodyBytes) > 0 {
		contentType = req.Header.Get("Content-Type")
	}

	host := req.URL.Host
	path := req.URL.EscapedPath()
	if path == "" {
		path = "/"
	}
	query := req.URL.RawQuery
	method := strings.ToUpper(req.Method)

	// Bitstamp v2 message - official format with spaces
	// "BITSTAMP" + " " + api_key + HTTP Verb + url.host + url.path + url.query + Content-Type + X-Auth-Nonce + X-Auth-Timestamp + X-Auth-Version + request.body
	message := "BITSTAMP " + t.APIKey +
		method +
		host +
		path +
		query +
		contentType +
		nonce +
		timestamp +
		"v2" +
		string(bodyBytes)

	sig := hex.EncodeToString(hmacSHA256(t.APISecret, []byte(message)))

	// Auth headers
	req.Header.Set("X-Auth", "BITSTAMP "+t.APIKey)
	req.Header.Set("X-Auth-Signature", sig)
	req.Header.Set("X-Auth-Nonce", nonce)
	req.Header.Set("X-Auth-Timestamp", timestamp)
	req.Header.Set("X-Auth-Version", "v2")

	// Send
	resp, err := rt.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Read/verify response signature (if provided), then restore body.
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return nil, err
	}
	_ = resp.Body.Close()

	if serverSig := resp.Header.Get("X-Server-Auth-Signature"); serverSig != "" {
		ct := resp.Header.Get("Content-Type")
		stringToSign := append([]byte(nonce+timestamp+ct), respBody...)
		check := hex.EncodeToString(hmacSHA256(t.APISecret, stringToSign))
		if serverSig != check {
			return nil, fmt.Errorf("bitstamp: server signature mismatch")
		}
	}

	// Restore for caller
	resp.Body = io.NopCloser(bytes.NewReader(respBody))
	resp.ContentLength = int64(len(respBody))
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(respBody)))

	return resp, nil
}

// ----- Helpers -----

func hmacSHA256(key, msg []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(msg)
	return h.Sum(nil)
}

// newNonce generates a unique 36-character nonce required by Bitstamp API.
// Must be unique within a 150-second window per API key.
func newNonce() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 36)
	rand.Read(b)
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}
