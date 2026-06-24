package generic

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAssetUMN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		asset             string
		expectedCurrency  string
		expectedPrecision int
		expectError       bool
		errorContains     string
	}{
		{
			name:              "USD/2",
			asset:             "USD/2",
			expectedCurrency:  "USD",
			expectedPrecision: 2,
			expectError:       false,
		},
		{
			name:              "BTC/8",
			asset:             "BTC/8",
			expectedCurrency:  "BTC",
			expectedPrecision: 8,
			expectError:       false,
		},
		{
			name:              "ETH/18",
			asset:             "ETH/18",
			expectedCurrency:  "ETH",
			expectedPrecision: 18,
			expectError:       false,
		},
		{
			name:              "JPY/0",
			asset:             "JPY/0",
			expectedCurrency:  "JPY",
			expectedPrecision: 0,
			expectError:       false,
		},
		{
			name:              "CUSTOM_COIN/6",
			asset:             "CUSTOM_COIN/6",
			expectedCurrency:  "CUSTOM_COIN",
			expectedPrecision: 6,
			expectError:       false,
		},
		// Assets without precision (defaults to 0)
		{
			name:              "COIN without precision",
			asset:             "COIN",
			expectedCurrency:  "COIN",
			expectedPrecision: 0,
			expectError:       false,
		},
		{
			name:              "JPY without precision",
			asset:             "JPY",
			expectedCurrency:  "JPY",
			expectedPrecision: 0,
			expectError:       false,
		},
		{
			name:              "TOKEN without precision",
			asset:             "TOKEN",
			expectedCurrency:  "TOKEN",
			expectedPrecision: 0,
			expectError:       false,
		},
		{
			name:              "USD without precision",
			asset:             "USD",
			expectedCurrency:  "USD",
			expectedPrecision: 0,
			expectError:       false,
		},
		{
			name:          "empty string",
			asset:         "",
			expectError:   true,
			errorContains: "empty asset",
		},
		{
			name:          "invalid precision",
			asset:         "USD/abc",
			expectError:   true,
			errorContains: "invalid precision",
		},
		{
			name:          "negative precision",
			asset:         "USD/-1",
			expectError:   true,
			errorContains: "must be non-negative",
		},
		{
			name:          "empty currency",
			asset:         "/2",
			expectError:   true,
			errorContains: "empty currency code",
		},
		{
			name:          "too many slashes",
			asset:         "USD/2/3",
			expectError:   true,
			errorContains: "invalid asset format",
		},
		// Lowercase / mixed-case currency codes must be rejected
		{
			name:          "lowercase currency with precision",
			asset:         "usd/2",
			expectError:   true,
			errorContains: "must be uppercase",
		},
		{
			name:          "mixed-case currency with precision",
			asset:         "Usd/2",
			expectError:   true,
			errorContains: "must be uppercase",
		},
		{
			name:          "lowercase currency without precision",
			asset:         "usd",
			expectError:   true,
			errorContains: "must be uppercase",
		},
		{
			name:          "mixed-case currency without precision",
			asset:         "Btc",
			expectError:   true,
			errorContains: "must be uppercase",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			currency, precision, err := parseAssetUMN(tc.asset)

			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorContains)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedCurrency, currency)
				require.Equal(t, tc.expectedPrecision, precision)
			}
		})
	}
}
