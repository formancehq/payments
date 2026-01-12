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
func parseAssetUMN(asset string) (currencyCode string, precision int, err error) {
	if asset == "" {
		return "", 0, fmt.Errorf("invalid asset format: empty asset")
	}

	parts := strings.Split(asset, "/")

	switch len(parts) {
	case 1:
		// No precision specified (e.g., "COIN", "JPY") - default to 0
		return parts[0], 0, nil

	case 2:
		// Precision specified (e.g., "USD/2", "BTC/8")
		currencyCode = parts[0]
		if currencyCode == "" {
			return "", 0, fmt.Errorf("invalid asset format: empty currency code")
		}

		precision, err = strconv.Atoi(parts[1])
		if err != nil {
			return "", 0, fmt.Errorf("invalid precision in asset %s: %w", asset, err)
		}

		if precision < 0 {
			return "", 0, fmt.Errorf("invalid precision in asset %s: must be non-negative", asset)
		}

		return currencyCode, precision, nil

	default:
		return "", 0, fmt.Errorf("invalid asset format: %s (expected CURRENCY or CURRENCY/PRECISION)", asset)
	}
}
