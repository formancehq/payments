package mangopay

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalAndValidateConfig(t *testing.T) {
	 t.Parallel()

	 tests := []struct {
		 name        string
		 payload     []byte
		 expected    Config
		 expectError bool
	 }{
		 {
			 name:    "Valid Config",
			 payload: []byte(`{"clientID":"client_123","apiKey":"sk_test","endpoint":"https://api.mangopay.com"}`),
			 expected: Config{
				 ClientID: "client_123",
				 APIKey:   "sk_test",
				 Endpoint: "https://api.mangopay.com",
			 },
			 expectError: false,
		 },
		 {
			 name:        "Missing Required Fields",
			 payload:     []byte(`{"clientID":"client_123"}`),
			 expected:    Config{},
			 expectError: true,
		 },
	 }

	 for _, tt := range tests {
		 t.Run(tt.name, func(t *testing.T) {
			 config, err := unmarshalAndValidateConfig(tt.payload)
			 if tt.expectError {
				 require.Error(t, err)
			 } else {
				 require.NoError(t, err)
				 assert.Equal(t, tt.expected, config)
			 }
		 })
	 }
}
