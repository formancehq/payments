package client

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// signingTransport centralises Kraken Pro auth as an http.RoundTripper
// (moneycorp pattern). Public endpoints (`/0/public/*`) pass through
// unsigned; private ones get a fresh nonce injected into the JSON body
// plus the `api-key` / `api-nonce` / `api-sign` headers. Keeping auth
// here leaves the request builders in client.go auth-agnostic and gives
// every retry attempt a fresh nonce + signature.
type signingTransport struct {
	apiKey    string
	apiSecret []byte
	nonce     atomic.Int64
	next      http.RoundTripper
}

func newSigningTransport(apiKey string, apiSecret []byte, next http.RoundTripper) *signingTransport {
	t := &signingTransport{apiKey: apiKey, apiSecret: apiSecret, next: next}
	// Seed with UnixNano so the first nonce sits above any prior
	// ms/us-precision caller: Kraken rejects any nonce <= the highest
	// ever used for the key.
	t.nonce.Store(time.Now().UnixNano())
	return t
}

// isPrivatePath reports whether a Kraken path requires signing.
func isPrivatePath(p string) bool { return strings.Contains(p, "/private/") }

func (t *signingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !isPrivatePath(req.URL.Path) {
		return t.next.RoundTrip(req)
	}
	nonce := t.nextNonce()
	body, err := bodyWithNonce(req, nonce)
	if err != nil {
		return nil, err
	}

	// Clone before mutating: the RoundTripper contract says not to modify
	// the caller's request.
	signed := req.Clone(req.Context())
	signed.Body = io.NopCloser(bytes.NewReader(body))
	signed.ContentLength = int64(len(body))
	signed.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	signed.Header.Set("Content-Type", "application/json")
	signed.Header.Set("api-key", t.apiKey)
	signed.Header.Set("api-nonce", nonce)
	signed.Header.Set("api-sign", sign(t.apiSecret, signed.URL.Path, nonce, body))
	return t.next.RoundTrip(signed)
}

// nextNonce returns max(prev+1, now) as an ASCII nonce. Kraken requires
// a strictly-increasing nonce per key; the atomic guard is the
// mutex-counter Kraken recommends for threads sharing a key — here the
// fetch_* activities that run concurrently against one client in a
// worker. Races between separate worker pods on the same key are
// unavoidable in-process; those surface as EAPI:Invalid nonce and are
// retried with backoff (see error.go, MAPPINGS).
func (t *signingTransport) nextNonce() string {
	for {
		prev := t.nonce.Load()
		next := prev + 1
		if now := time.Now().UnixNano(); now > next {
			next = now
		}
		if t.nonce.CompareAndSwap(prev, next) {
			return strconv.FormatInt(next, 10)
		}
	}
}

// bodyWithNonce returns the request's JSON body with the nonce injected.
// Kraken's signature covers the body, so the nonce must be in the bytes
// actually sent (the `api-nonce` header alone is not enough).
func bodyWithNonce(req *http.Request, nonce string) ([]byte, error) {
	params := map[string]any{}
	if req.Body != nil {
		raw, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &params); err != nil {
				return nil, fmt.Errorf("decode request body: %w", err)
			}
		}
	}
	params["nonce"] = nonce
	return json.Marshal(params)
}

// sign computes the Kraken API-Sign per docs:
//
//	base64( HMAC-SHA512( secret, uriPath || SHA256(nonce || body) ) )
func sign(secret []byte, uriPath, nonce string, body []byte) string {
	sha := sha256.New()
	sha.Write([]byte(nonce))
	sha.Write(body)
	mac := hmac.New(sha512.New, secret)
	mac.Write([]byte(uriPath))
	mac.Write(sha.Sum(nil))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
