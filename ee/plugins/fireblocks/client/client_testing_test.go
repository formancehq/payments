package client

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

// newTestClient builds a Fireblocks client pointed at baseURL using a freshly
// generated RSA key. The httptest server does not validate the JWT, so any
// well-formed key works.
func newTestClient(t *testing.T, baseURL string) Client {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	return New("fireblocks-test", "test-api-key", key, baseURL)
}
