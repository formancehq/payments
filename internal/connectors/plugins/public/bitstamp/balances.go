package bitstamp

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	// Fetch balances from Bitstamp
	balancesResp, err := p.client.GetBalances(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to get balances: %w", err)
	}

	balances := make([]models.PSPBalance, 0, len(balancesResp))
	for _, bal := range balancesResp {
		// Skip zero balances
		if bal.Balance == "0" || bal.Balance == "0.00000000" || bal.Balance == "" {
			continue
		}

		balance, err := parseBitstampBalance(bal.Currency, bal.Balance)
		if err != nil {
			p.logger.Errorf("failed to parse balance for %s: %v", bal.Currency, err)
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

func parseBitstampBalance(currency string, amountStr string) (models.PSPBalance, error) {
	if amountStr == "" {
		amountStr = "0"
	}

	// Normalize currency (Bitstamp uses lowercase)
	normalizedCurrency := strings.ToUpper(currency)

	// Get precision for the asset
	precision := GetPrecision(normalizedCurrency)

	// Parse amount with flexible precision
	amount, err := parseAmountWithPrecision(amountStr, precision)
	if err != nil {
		return models.PSPBalance{}, fmt.Errorf("failed to parse amount %s: %w", amountStr, err)
	}

	return models.PSPBalance{
		AccountReference: "main",
		Asset:            normalizedCurrency,
		Amount:           amount,
		CreatedAt:        time.Now(),
	}, nil
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

func parseBitstampAmount(amountStr string, asset string) (*big.Int, error) {
	if amountStr == "" {
		return big.NewInt(0), nil
	}

	precision := GetPrecision(asset)
	return parseAmountWithPrecision(amountStr, precision)
}

// parseDecimalString parses a decimal string to big.Int with a given precision
func parseDecimalString(s string, precision int) (*big.Int, error) {
	return parseAmountWithPrecision(s, precision)
}
