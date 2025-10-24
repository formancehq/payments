package modulr

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
			 payload: []byte(`{"apiKey":"ak_test","apiSecret":"as_test","endpoint":"https://api.modulr.com"}`),
			 expected: Config{
				 APIKey:        "ak_test",
				 APISecret:     "as_test",
				 Endpoint:      "https://api.modulr.com",
				 PollingPeriod: defaultPollingPeriod,
			 },
			 expectError: false,
		 },
		 {
			 name:        "Missing Required Fields",
			 payload:     []byte(`{"apiKey":"ak_test"}`),
			 expected:    Config{},
			 expectError: true,
		 },
		 {
			 name:    "Non default polling period",
			 payload: []byte(`{"apiKey":"ak_test","apiSecret":"as_test","endpoint":"https://api.modulr.com","pollingPeriod":"45m"}`),
			 expected: Config{
				 APIKey:        "ak_test",
				 APISecret:     "as_test",
				 Endpoint:      "https://api.modulr.com",
				 PollingPeriod: longPollingPeriod,
			 },
			 expectError: false,
		 },
		 {
			 name:        "Invalid polling period",
			 payload:     []byte(`{"apiKey":"ak_test","apiSecret":"as_test","endpoint":"https://api.modulr.com","pollingPeriod":"not-a-duration"}`),
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
