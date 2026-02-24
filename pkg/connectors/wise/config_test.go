package wise

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/pkg/connector"
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

	 defaultPollingPeriod, _ := connector.NewPollingPeriod("", connector.DefaultPollingPeriod, connector.MinimumPollingPeriod)
	 longPollingPeriod, _ := connector.NewPollingPeriod("45m", connector.DefaultPollingPeriod, connector.MinimumPollingPeriod)

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
				 PollingPeriod:    defaultPollingPeriod,
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
			 name:    "Non default polling period",
			 payload: makePayload(t, map[string]string{"apiKey": "sk_test", "webhookPublicKey": validRSAPublicKeyPEM, "pollingPeriod": "45m"}),
			 expected: Config{
				 APIKey:           "sk_test",
				 WebhookPublicKey: validRSAPublicKeyPEM,
				 PollingPeriod:    longPollingPeriod,
			 },
			 expectError: false,
		 },
		 {
			 name:        "Invalid polling period",
			 payload:     makePayload(t, map[string]string{"apiKey": "sk_test", "webhookPublicKey": validRSAPublicKeyPEM, "pollingPeriod": "not-a-duration"}),
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
				 // For invalid polling period and invalid key, ensure error is marked as invalid config
				 if tt.name == "Invalid polling period" || tt.name == "Invalid public key" {
					 assert.ErrorContains(t, err, connector.ErrInvalidConfig.Error())
				 }
			 } else {
				 require.NoError(t, err)
				 assert.Equal(t, tt.expected.APIKey, config.APIKey)
				 assert.Equal(t, tt.expected.WebhookPublicKey, config.WebhookPublicKey)
				 assert.Equal(t, tt.expected.PollingPeriod, config.PollingPeriod)
			 }
		 })
	 }
}
