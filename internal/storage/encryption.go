package storage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// EncryptRaw encrypts a JSON payload using Postgres pgcrypto and the storage encryption key.
// It mirrors the encryption performed in other storage methods (e.g., connectors install).
func (s *store) EncryptRaw(message json.RawMessage) (json.RawMessage, error) {
	// Use a simple SELECT to leverage pgp_sym_encrypt with consistent options
	// We encrypt the JSON as TEXT to match existing patterns
	var cipher []byte
	// bun.NewRaw with positional args; we cast to text in the SQL expression
	if err := s.db.NewRaw("SELECT pgp_sym_encrypt(?::TEXT, ?::TEXT, ?::TEXT)", string(message), s.configEncryptionKey, encryptionOptions).Scan(context.Background(), &cipher); err != nil {
		return nil, err
	}
	// Base64-encode the binary ciphertext so it can be safely marshaled as JSON
	b64 := base64.StdEncoding.EncodeToString(cipher)
	return []byte(fmt.Sprintf("%q", b64)), nil
}

// DecryptRaw decrypts a JSON payload previously encrypted with encryptRaw
func (s *store) DecryptRaw(message json.RawMessage) (json.RawMessage, error) {
	// Expect a JSON string containing base64-encoded ciphertext
	var b64 string
	if err := json.Unmarshal(message, &b64); err != nil {
		return nil, err
	}
	cipher, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}

	var plain string
	if err := s.db.NewRaw("SELECT pgp_sym_decrypt(?::BYTEA, ?::TEXT, ?::TEXT)", cipher, s.configEncryptionKey, encryptionOptions).Scan(context.Background(), &plain); err != nil {
		return nil, err
	}
	return json.RawMessage(plain), nil
}
