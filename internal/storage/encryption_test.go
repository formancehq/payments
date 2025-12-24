package storage

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptRaw(t *testing.T) {
	t.Parallel()

	st := newStore(t)

	plain := json.RawMessage(`{"foo":"bar","n":123}`)

	cipher, err := st.EncryptRaw(plain)
	require.NoError(t, err)
	require.NotNil(t, cipher)
	require.NotEqual(t, string(plain), string(cipher))

	// Ensure cipher is a JSON string and is valid base64
	var b64 string
	require.NoError(t, json.Unmarshal(cipher, &b64))
	_, err = base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err)

	back, err := st.DecryptRaw(cipher)
	require.NoError(t, err)
	require.Equal(t, string(plain), string(back))
}
