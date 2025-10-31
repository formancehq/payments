package atlar

import (
	"testing"

	"github.com/formancehq/payments/internal/connectors/plugins/sharedconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalAndValidateConfig(t *testing.T) {
	 t.Parallel()

	 defaultPollingPeriod, _ := sharedconfig.NewPollingPeriod("", sharedconfig.DefaultPollingPeriod, sharedconfig.MinimumPollingPeriod)
	 longPollingPeriod, _ := sharedconfig.NewPollingPeriod("45m", sharedconfig.DefaultPollingPeriod, sharedconfig.MinimumPollingPeriod)

	 tests := []struct {
		 name        string
		 payload     []byte
		 expected    Config
		 expectError bool
	 }{
		 {
			 name:    "Valid Config",
			 payload: []byte(`{"baseUrl":"https://api.atlar.com","accessKey":"ak_test","secret":"sk_test"}`),
			 expected: Config{
				 BaseURL:       "https://api.atlar.com",
				 AccessKey:     "ak_test",
				 Secret:        "sk_test",
				 PollingPeriod: defaultPollingPeriod,
			 },
			 expectError: false,
		 },
		 {
			 name:        "Missing Required Fields",
			 payload:     []byte(`{"baseUrl":"https://api.atlar.com"}`),
			 expected:    Config{},
			 expectError: true,
		 },
		 {
			 name:    "Non default polling period",
			 payload: []byte(`{"baseUrl":"https://api.atlar.com","accessKey":"ak_test","secret":"sk_test","pollingPeriod":"45m"}`),
			 expected: Config{
				 BaseURL:       "https://api.atlar.com",
				 AccessKey:     "ak_test",
				 Secret:        "sk_test",
				 PollingPeriod: longPollingPeriod,
			 },
			 expectError: false,
		 },
		 {
			 name:        "Invalid polling period",
			 payload:     []byte(`{"baseUrl":"https://api.atlar.com","accessKey":"ak_test","secret":"sk_test","pollingPeriod":"not-a-duration"}`),
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
