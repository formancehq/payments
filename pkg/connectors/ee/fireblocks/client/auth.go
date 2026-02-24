package client

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type fireblocksTransport struct {
	apiKey     string
	privateKey *rsa.PrivateKey
	underlying http.RoundTripper
}

func (t *fireblocksTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	bodyHash := sha256.Sum256(bodyBytes)
	bodyHashHex := hex.EncodeToString(bodyHash[:])

	uri := req.URL.Path
	if req.URL.RawQuery != "" {
		uri = uri + "?" + req.URL.RawQuery
	}

	token, err := t.generateJWT(uri, bodyHashHex)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	req.Header.Set("X-API-Key", t.apiKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	return t.underlying.RoundTrip(req)
}

func (t *fireblocksTransport) generateJWT(uri, bodyHash string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"uri":      uri,
		"nonce":    uuid.New().String(),
		"iat":      now.Unix(),
		"exp":      now.Add(30 * time.Second).Unix(),
		"sub":      t.apiKey,
		"bodyHash": bodyHash,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(t.privateKey)
}
