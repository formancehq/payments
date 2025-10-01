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

// Client describes the minimal interface your connector can depend on.
type Client interface {
	GetTransactions(ctx context.Context, params TransactionsParams) ([]Transaction, error)
	GetAccounts(ctx context.Context, page, pageSize int) ([]*Account, error)
	GetAccountBalances(ctx context.Context) ([]*Balance, error)
}

type client struct {
	apiKey    string
	apiSecret []byte

	baseURL    *url.URL
	httpClient *http.Client
	underlying http.RoundTripper
	timeout    time.Duration
}

// New creates a Bitstamp client with a signing transport.
func New(apiKey string, apiSecret []byte, opts ...Option) Client {
	c := &client{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		baseURL:   mustParseURL("https://www.bitstamp.net"),
	}

	for _, opt := range opts {
		opt(c)
	}

	tr := &signingTransport{
		APIKey:     c.apiKey,
		APISecret:  c.apiSecret,
		Underlying: c.underlying, // may be nil → http.DefaultTransport
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{
			Transport: tr,
			Timeout:   c.timeoutOrDefault(),
		}
	} else {
		// Ensure our signing transport wraps the provided client
		if c.httpClient.Transport == nil {
			c.httpClient.Transport = tr
		} else {
			// If they already set a transport, we wrap it as the Underlying
			tr.Underlying = c.httpClient.Transport
			c.httpClient.Transport = tr
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

// ----- Implementation -----

func (c *client) timeoutOrDefault() time.Duration {
	if c.timeout > 0 {
		return c.timeout
	}
	return 30 * time.Second
}

// ----- Signing Transport (Bitstamp v2) -----

type signingTransport struct {
	APIKey     string
	APISecret  []byte
	Underlying http.RoundTripper
}

func (t *signingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt := t.Underlying
	if rt == nil {
		rt = http.DefaultTransport
	}

	// Timestamp (ms) and nonce per request
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
	if bodyBytes != nil {
		req.ContentLength = int64(len(bodyBytes))
	}

	// Content-Type used in the signature. If header absent, it's an empty string.
	contentType := req.Header.Get("Content-Type")

	host := req.URL.Host
	path := req.URL.EscapedPath()
	if path == "" {
		path = "/"
	}
	query := req.URL.RawQuery
	method := strings.ToUpper(req.Method)

	// Bitstamp v2 message
	message :=
		"BITSTAMP " + t.APIKey +
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

func newNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}
