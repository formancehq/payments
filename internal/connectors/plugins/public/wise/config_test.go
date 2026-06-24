package wise

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validRSAPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBALiJyoSgGJE0E7E5Wdl66iRS0LlwM651
01qmPvvrLpzjAU6YewsGmmKzBSSMSmc5QwDFi1Cdm42Hcps225y7sKsCAwEAAQ==
-----END PUBLIC KEY-----`

func makePayload(t *testing.T, v any) []byte {
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

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
			 payload: makePayload(t, map[string]string{"apiKey": "sk_test", "webhookPublicKey": validRSAPublicKeyPEM}),
			 expected: Config{
				 APIKey:           "sk_test",
				 WebhookPublicKey: validRSAPublicKeyPEM,
			 },
			 expectError: false,
		 },
		 {
			 name:        "Missing Required Fields",
			 payload:     makePayload(t, map[string]string{"apiKey": "sk_test"}),
			 expected:    Config{},
			 expectError: true,
		 },
		 {
			 name:        "Invalid public key",
			 payload:     makePayload(t, map[string]string{"apiKey": "sk_test", "webhookPublicKey": "not-a-valid-pem"}),
			 expected:    Config{},
			 expectError: true,
		 },
	 }

	 for _, tt := range tests {
		 t.Run(tt.name, func(t *testing.T) {
			 config, err := unmarshalAndValidateConfig(tt.payload)
			 if tt.expectError {
				 require.Error(t, err)
				 if tt.name == "Invalid public key" {
					 assert.ErrorContains(t, err, models.ErrInvalidConfig.Error())
				 }
			 } else {
				 require.NoError(t, err)
				 assert.Equal(t, tt.expected.APIKey, config.APIKey)
				 assert.Equal(t, tt.expected.WebhookPublicKey, config.WebhookPublicKey)
			 }
		 })
	 }
}
