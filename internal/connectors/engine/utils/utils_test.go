package utils

import (
	"errors"
	"testing"

	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFormanceRedirectURLQueryValues(t *testing.T) {
	t.Parallel()

	const stackPublicURL = "https://my-stack.example.com"
	connectorID := models.ConnectorID{Reference: uuid.New(), Provider: "plaid"}
	otherConnectorID := models.ConnectorID{Reference: uuid.New(), Provider: "plaid"}

	canonical, err := GetFormanceRedirectURL(stackPublicURL, connectorID)
	require.NoError(t, err)

	canonicalForOtherConnector, err := GetFormanceRedirectURL(stackPublicURL, otherConnectorID)
	require.NoError(t, err)

	tests := map[string]struct {
		queryValues map[string][]string
		expectError bool
	}{
		"absent - nothing to validate": {
			queryValues: map[string][]string{},
		},
		"nil query values": {
			queryValues: nil,
		},
		"matches this connector's exact canonical path": {
			queryValues: map[string][]string{
				FormanceRedirectURLQueryParamID: {canonical},
			},
		},
		"same origin as this connector's canonical path, but a different path - rejected": {
			// A forged/replayed webhook pointing at some other same-origin
			// path (which could itself redirect elsewhere) must not pass
			// just because the host matches.
			queryValues: map[string][]string{
				FormanceRedirectURLQueryParamID: {stackPublicURL + "/some-other-endpoint"},
			},
			expectError: true,
		},
		"another connector's otherwise-valid canonical path - rejected": {
			// Matching *a* legitimate-looking open-banking redirect path
			// isn't enough - it must be THIS connector's.
			queryValues: map[string][]string{
				FormanceRedirectURLQueryParamID: {canonicalForOtherConnector},
			},
			expectError: true,
		},
		"different host - rejected": {
			queryValues: map[string][]string{
				FormanceRedirectURLQueryParamID: {"https://evil.example.com/steal"},
			},
			expectError: true,
		},
		"different scheme - rejected": {
			queryValues: map[string][]string{
				FormanceRedirectURLQueryParamID: {"http://" + canonical[len("https://"):]},
			},
			expectError: true,
		},
		"ambiguous - more than one value - rejected": {
			queryValues: map[string][]string{
				FormanceRedirectURLQueryParamID: {canonical, canonical},
			},
			expectError: true,
		},
		"present but empty - rejected": {
			queryValues: map[string][]string{
				FormanceRedirectURLQueryParamID: {},
			},
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := ValidateFormanceRedirectURLQueryValues(stackPublicURL, connectorID, tt.queryValues)
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

	connectorID := models.ConnectorID{Reference: uuid.New(), Provider: "plaid"}

	// A malformed stackPublicURL is our own misconfiguration, not an
	// untrusted caller value - it must still be rejected, but it
	// deliberately isn't classified as ErrInvalidFormanceRedirectURL.
	err := ValidateFormanceRedirectURLQueryValues("not a url\x7f", connectorID, map[string][]string{
		FormanceRedirectURLQueryParamID: {"https://my-stack.example.com/redirect"},
	})
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrInvalidFormanceRedirectURL))
}
