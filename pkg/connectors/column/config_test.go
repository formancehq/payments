package column

import (
	"testing"

	"github.com/formancehq/payments/pkg/connector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			 payload: []byte(`{"apiKey":"sk_test","endpoint":"https://api.column.com"}`),
			 expected: Config{
				 APIKey:        "sk_test",
				 Endpoint:      "https://api.column.com",
				 PollingPeriod: defaultPollingPeriod,
			 },
			 expectError: false,
		 },
		 {
			 name:        "Missing Required Fields",
			 payload:     []byte(`{"apiKey":"sk_test"}`),
			 expected:    Config{},
			 expectError: true,
		 },
		 {
			 name:    "Non default polling period",
			 payload: []byte(`{"apiKey":"sk_test","endpoint":"https://api.column.com","pollingPeriod":"45m"}`),
			 expected: Config{
				 APIKey:        "sk_test",
				 Endpoint:      "https://api.column.com",
				 PollingPeriod: longPollingPeriod,
			 },
			 expectError: false,
		 },
		 {
			 name:        "Invalid polling period",
			 payload:     []byte(`{"apiKey":"sk_test","endpoint":"https://api.column.com","pollingPeriod":"not-a-duration"}`),
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
