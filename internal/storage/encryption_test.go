package storage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptRaw(t *testing.T) {
	t.Parallel()

	st := newStore(t)

	plain := json.RawMessage(`{"foo":"bar","n":123}`)

	cipher, err := st.EncryptRaw(context.Background(), plain)
	require.NoError(t, err)
	require.NotNil(t, cipher)
	require.NotEqual(t, string(plain), string(cipher))

	// Ensure cipher is a JSON string and is valid base64
	var b64 string
	require.NoError(t, json.Unmarshal(cipher, &b64))
	_, err = base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err)

	back, err := st.DecryptRaw(context.Background(), cipher)
	require.NoError(t, err)
	require.Equal(t, string(plain), string(back))
}

func TestDecryptRaw_NotJSONString(t *testing.T) {
	t.Parallel()

	st := newStore(t)

	// Provide a JSON object instead of a JSON string → should be treated as not encrypted
	notString := json.RawMessage(`{"foo":"bar"}`)

	back, err := st.DecryptRaw(context.Background(), notString)
	require.ErrorIs(t, err, ErrNotEncrypted)
	require.Nil(t, back)
}

func TestDecryptRaw_InvalidBase64String(t *testing.T) {
	t.Parallel()

	st := newStore(t)

	// Provide a JSON string that is not valid base64 → should be treated as not encrypted
	invalidB64 := json.RawMessage(`"not_base64_$$$"`)

	back, err := st.DecryptRaw(context.Background(), invalidB64)
	require.ErrorIs(t, err, ErrNotEncrypted)
	require.Nil(t, back)
}
