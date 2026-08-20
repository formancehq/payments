package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFormanceRedirectURLQueryValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stackPublicURL string
		queryValues    map[string][]string
		expectError    bool
	}{
		"absent - nothing to validate": {
			stackPublicURL: "https://my-stack.example.com",
			queryValues:    map[string][]string{},
		},
		"nil query values": {
			stackPublicURL: "https://my-stack.example.com",
			queryValues:    nil,
		},
		"matches stack public URL": {
			stackPublicURL: "https://my-stack.example.com",
			queryValues: map[string][]string{
				FormanceRedirectURLQueryParamID: {"https://my-stack.example.com/api/payments/v3/connectors/open-banking/abc/redirect"},
			},
		},
		"different host - rejected": {
			stackPublicURL: "https://my-stack.example.com",
			queryValues: map[string][]string{
				FormanceRedirectURLQueryParamID: {"https://evil.example.com/steal"},
			},
			expectError: true,
		},
		"different scheme - rejected": {
			stackPublicURL: "https://my-stack.example.com",
			queryValues: map[string][]string{
				FormanceRedirectURLQueryParamID: {"http://my-stack.example.com/api/payments/v3/connectors/open-banking/abc/redirect"},
			},
			expectError: true,
		},
		"malformed URL - rejected": {
			stackPublicURL: "https://my-stack.example.com",
			queryValues: map[string][]string{
				FormanceRedirectURLQueryParamID: {"not a url\x7f"},
			},
			expectError: true,
		},
		"ambiguous - more than one value - rejected": {
			stackPublicURL: "https://my-stack.example.com",
			queryValues: map[string][]string{
				FormanceRedirectURLQueryParamID: {
					"https://my-stack.example.com/a",
					"https://my-stack.example.com/b",
				},
			},
			expectError: true,
		},
		"present but empty - rejected": {
			stackPublicURL: "https://my-stack.example.com",
			queryValues: map[string][]string{
				FormanceRedirectURLQueryParamID: {},
			},
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := ValidateFormanceRedirectURLQueryValues(tt.stackPublicURL, tt.queryValues)
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidFormanceRedirectURL)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateFormanceRedirectURLQueryValues_MalformedStackPublicURL(t *testing.T) {
	t.Parallel()

	// A malformed stackPublicURL is our own misconfiguration, not an
	// untrusted caller value - it must still be rejected, but it
	// deliberately isn't classified as ErrInvalidFormanceRedirectURL.
	err := ValidateFormanceRedirectURLQueryValues("not a url\x7f", map[string][]string{
		FormanceRedirectURLQueryParamID: {"https://my-stack.example.com/redirect"},
	})
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrInvalidFormanceRedirectURL))
}
