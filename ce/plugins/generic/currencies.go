package generic

import (
	"fmt"
	"strconv"
	"strings"
)

// parseAssetUMN parses an asset in UMN format.
// Supports two formats:
//   - With precision: "USD/2", "BTC/8", "COIN/6"
//   - Without precision: "COIN", "JPY", "TOKEN" (defaults to precision 0)
//
// Returns the currency code and precision.
// This allows the generic connector to accept ANY asset.
func parseAssetUMN(asset string) (string, int, error) {
	if asset == "" {
		return "", 0, fmt.Errorf("invalid asset format: empty asset")
	}

	parts := strings.Split(asset, "/")

	switch len(parts) {
	case 1:
		if err := validateCurrencyCode(parts[0], asset); err != nil {
			return "", 0, err
		}
		return parts[0], 0, nil

	case 2:
		if err := validateCurrencyCode(parts[0], asset); err != nil {
			return "", 0, err
		}

		precision, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", 0, fmt.Errorf("invalid precision in asset %s: %w", asset, err)
		}
		if precision < 0 {
			return "", 0, fmt.Errorf("invalid precision in asset %s: must be non-negative", asset)
		}

		return parts[0], precision, nil

	default:
		return "", 0, fmt.Errorf("invalid asset format: %s (expected CURRENCY or CURRENCY/PRECISION)", asset)
	}
}

func validateCurrencyCode(code, asset string) error {
	if code == "" {
		return fmt.Errorf("invalid asset format: empty currency code")
	}
	if code != strings.ToUpper(code) {
		return fmt.Errorf("invalid asset %s: currency code must be uppercase", asset)
	}
	return nil
}
