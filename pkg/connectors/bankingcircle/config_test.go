package bankingcircle

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
			 payload: []byte(`{"username":"user","password":"pass","endpoint":"https://api.bankingcircle.com","authorizationEndpoint":"https://auth.bankingcircle.com","userCertificate":"cert","userCertificateKey":"key"}`),
			 expected: Config{
				 Username:              "user",
				 Password:              "pass",
				 Endpoint:              "https://api.bankingcircle.com",
				 AuthorizationEndpoint: "https://auth.bankingcircle.com",
				 UserCertificate:       "cert",
				 UserCertificateKey:    "key",
				 PollingPeriod:         defaultPollingPeriod,
			 },
			 expectError: false,
		 },
		 {
			 name:        "Missing Required Fields",
			 payload:     []byte(`{"username":"user"}`),
			 expected:    Config{},
			 expectError: true,
		 },
		 {
			 name:    "Non default polling period",
			 payload: []byte(`{"username":"user","password":"pass","endpoint":"https://api.bankingcircle.com","authorizationEndpoint":"https://auth.bankingcircle.com","userCertificate":"cert","userCertificateKey":"key","pollingPeriod":"45m"}`),
			 expected: Config{
				 Username:              "user",
				 Password:              "pass",
				 Endpoint:              "https://api.bankingcircle.com",
				 AuthorizationEndpoint: "https://auth.bankingcircle.com",
				 UserCertificate:       "cert",
				 UserCertificateKey:    "key",
				 PollingPeriod:         longPollingPeriod,
			 },
			 expectError: false,
		 },
		 {
			 name:        "Invalid polling period",
			 payload:     []byte(`{"username":"user","password":"pass","endpoint":"https://api.bankingcircle.com","authorizationEndpoint":"https://auth.bankingcircle.com","userCertificate":"cert","userCertificateKey":"key","pollingPeriod":"not-a-duration"}`),
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
