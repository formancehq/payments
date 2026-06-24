package bankingcircle

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
			 payload: []byte(`{"username":"user","password":"pass","endpoint":"https://api.bankingcircle.com","authorizationEndpoint":"https://auth.bankingcircle.com","userCertificate":"cert","userCertificateKey":"key"}`),
			 expected: Config{
				 Username:              "user",
				 Password:              "pass",
				 Endpoint:              "https://api.bankingcircle.com",
				 AuthorizationEndpoint: "https://auth.bankingcircle.com",
				 UserCertificate:       "cert",
				 UserCertificateKey:    "key",
			 },
			 expectError: false,
		 },
		 {
			 name:        "Missing Required Fields",
			 payload:     []byte(`{"username":"user"}`),
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
