package adyen

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
			payload: []byte(`{"apiKey":"123","companyID":"456","liveEndpointPrefix":"live"}`),
			expected: Config{
				APIKey:             "123",
				CompanyID:          "456",
				LiveEndpointPrefix: "live",
			},
			expectError: false,
		},
		{
			name:        "Missing Required Fields",
			payload:     []byte(`{"liveEndpointPrefix":"live"}`),
			expected:    Config{},
			expectError: true,
		},
		{
			name:    "LiveEndpointPrefix with hyphen",
			payload: []byte(`{"apiKey":"123","companyID":"456","liveEndpointPrefix":"1797a841fbb37ca7-AdyenDemo"}`),
			expected: Config{
				APIKey:             "123",
				CompanyID:          "456",
				LiveEndpointPrefix: "1797a841fbb37ca7-AdyenDemo",
			},
			expectError: false,
		},
		{
			name:        "LiveEndpointPrefix with url unsafe values",
			payload:     []byte(`{"apiKey":"123","companyID":"456","liveEndpointPrefix":"live%jksj"}`),
			expected:    Config{},
			expectError: true,
		},
		{
			name:    "Valid Optional Fields",
			payload: []byte(`{"apiKey":"123","companyID":"456","webhookUsername":"user","webhookPassword":"pass"}`),
			expected: Config{
				APIKey:          "123",
				CompanyID:       "456",
				WebhookUsername: "user",
				WebhookPassword: "pass",
			},
			expectError: false,
		},
		{
			name:        "Invalid WebhookUsername",
			payload:     []byte(`{"apiKey":"123","companyID":"456","webhookUsername":"user:invalid"}`),
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
