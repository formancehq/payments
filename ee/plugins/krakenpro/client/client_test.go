package client

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignRequest(t *testing.T) {
	// Known test vector for Kraken HMAC-SHA512 signing.
	// Private key (base64): "kQH5HW/8p1uGOVjbgWA7FunAmGO8lsSUXNsu3eow76sz84Q18fWxnyRzBHCd3pd5nE9qa99HAZtuZuj6F1huXg=="
	// This is a well-known test from Kraken's documentation examples.
	privateKeyB64 := "kQH5HW/8p1uGOVjbgWA7FunAmGO8lsSUXNsu3eow76sz84Q18fWxnyRzBHCd3pd5nE9qa99HAZtuZuj6F1huXg=="

	decodedKey, err := base64.StdEncoding.DecodeString(privateKeyB64)
	require.NoError(t, err)

	c := &client{
		apiKey:     "test-api-key",
		privateKey: decodedKey,
	}

	// Test that signing produces a non-empty base64 string
	uriPath := "/0/private/Balance"
	var nonce int64 = 1616492376594
	postData := "nonce=1616492376594"

	signature := c.signRequest(uriPath, nonce, postData)

	// Verify it's valid base64
	_, err = base64.StdEncoding.DecodeString(signature)
	assert.NoError(t, err)
	assert.NotEmpty(t, signature)

	// Verify deterministic: same inputs produce same output
	signature2 := c.signRequest(uriPath, nonce, postData)
	assert.Equal(t, signature, signature2)

	// Verify different nonce produces different signature
	signature3 := c.signRequest(uriPath, nonce+1, "nonce=1616492376595")
	assert.NotEqual(t, signature, signature3)
}

func TestKrakenError(t *testing.T) {
	err := &KrakenError{Errors: []string{"EAPI:Invalid key", "EGeneral:Internal error"}}
	assert.Equal(t, "EAPI:Invalid key; EGeneral:Internal error", err.Error())
	assert.False(t, err.IsRateLimited())

	rateLimitErr := &KrakenError{Errors: []string{"EAPI:Rate limit exceeded"}}
	assert.True(t, rateLimitErr.IsRateLimited())
}

func TestNewClientInvalidKey(t *testing.T) {
	_, err := New("test", "api-key", "not-valid-base64!!!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode private key")
}
