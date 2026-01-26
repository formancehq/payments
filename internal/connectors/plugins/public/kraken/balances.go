package kraken

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/assets"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	// Fetch balances from Kraken
	balancesResp, err := p.client.GetBalance(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to get balances: %w", err)
	}

	balances := make([]models.PSPBalance, 0, len(balancesResp))
	for asset, amountStr := range balancesResp {
		balance, err := parseKrakenBalance(asset, amountStr)
		if err != nil {
			p.logger.Errorf("failed to parse balance for %s: %v", asset, err)
			continue
		}
		balances = append(balances, balance)
	}

	return models.FetchNextBalancesResponse{
		Balances: balances,
		NewState: nil,
		HasMore:  false,
	}, nil
}

func parseKrakenBalance(asset string, amountStr string) (models.PSPBalance, error) {
	if amountStr == "" {
		amountStr = "0"
	}

	// Normalize Kraken asset names
	normalizedAsset := normalizeKrakenAsset(asset)

	// Validate the normalized asset
	if !assets.IsValid(normalizedAsset) {
		return models.PSPBalance{}, fmt.Errorf("invalid asset %q (normalized: %q): does not match required format", asset, normalizedAsset)
	}

	// Get precision for the asset
	precision := GetPrecision(normalizedAsset)

	// Parse amount with flexible precision (Kraken may return more decimals than expected)
	amount, err := parseAmountWithPrecision(amountStr, precision)
	if err != nil {
		return models.PSPBalance{}, fmt.Errorf("failed to parse amount %s: %w", amountStr, err)
	}

	return models.PSPBalance{
		AccountReference: "main",
		Asset:            normalizedAsset,
		Amount:           amount,
		CreatedAt:        time.Now(),
	}, nil
}

// normalizeKrakenAsset converts Kraken's internal asset names to standard names.
// It handles Kraken's special prefixes and converts to uppercase.
func normalizeKrakenAsset(asset string) string {
	// Kraken uses special prefixes for some assets
	// X prefix for crypto (e.g., XXBT = BTC)
	// Z prefix for fiat (e.g., ZEUR = EUR)
	krakenToStandard := map[string]string{
		"XXBT":  "BTC",
		"XBT":   "BTC",
		"XETH":  "ETH",
		"XLTC":  "LTC",
		"XXRP":  "XRP",
		"XXLM":  "XLM",
		"XDOGE": "DOGE",
		"ZEUR":  "EUR",
		"ZUSD":  "USD",
		"ZGBP":  "GBP",
		"ZJPY":  "JPY",
		"ZCAD":  "CAD",
		"ZAUD":  "AUD",
		"ZCHF":  "CHF",
	}

	if standardName, ok := krakenToStandard[asset]; ok {
		return standardName
	}

	// Convert to uppercase for assets not in the mapping
	return strings.ToUpper(strings.TrimSpace(asset))
}

func parseKrakenAmount(amountStr string, asset string) (*big.Int, error) {
	if amountStr == "" {
		return big.NewInt(0), nil
	}

	precision := GetPrecision(asset)
	return parseAmountWithPrecision(amountStr, precision)
}

// parseAmountWithPrecision parses an amount string and converts it to the smallest unit
// with the given precision. It handles cases where the input has more decimal places
// than the target precision by truncating (not rounding).
func parseAmountWithPrecision(amountStr string, precision int) (*big.Int, error) {
	if amountStr == "" {
		return big.NewInt(0), nil
	}

	// Split into integer and fractional parts
	parts := strings.Split(amountStr, ".")
	intPart := parts[0]
	fracPart := ""
	if len(parts) > 1 {
		fracPart = parts[1]
	}

	// Truncate or pad fractional part to target precision
	if len(fracPart) > precision {
		fracPart = fracPart[:precision]
	} else {
		for len(fracPart) < precision {
			fracPart += "0"
		}
	}

	// Combine into a single string without decimal point
	combined := intPart + fracPart

	// Remove leading zeros (but keep at least one digit)
	combined = strings.TrimLeft(combined, "0")
	if combined == "" {
		combined = "0"
	}

	// Parse as big.Int
	amount := new(big.Int)
	_, ok := amount.SetString(combined, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse amount: %s", amountStr)
	}

	return amount, nil
}
