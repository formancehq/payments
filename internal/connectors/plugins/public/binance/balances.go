package binance

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	// Fetch account info which includes balances
	accountInfo, err := p.client.GetAccountInfo(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to get account info: %w", err)
	}

	balances := make([]models.PSPBalance, 0, len(accountInfo.Balances))
	for _, bal := range accountInfo.Balances {
		// Skip zero balances (both free and locked)
		if (bal.Free == "0" || bal.Free == "0.00000000") && (bal.Locked == "0" || bal.Locked == "0.00000000") {
			continue
		}

		// Calculate total balance (free + locked)
		freeAmount, err := parseBinanceAmount(bal.Free, bal.Asset)
		if err != nil {
			p.logger.Errorf("failed to parse free balance for %s: %v", bal.Asset, err)
			continue
		}

		lockedAmount, err := parseBinanceAmount(bal.Locked, bal.Asset)
		if err != nil {
			p.logger.Errorf("failed to parse locked balance for %s: %v", bal.Asset, err)
			continue
		}

		totalAmount := new(big.Int).Add(freeAmount, lockedAmount)

		balances = append(balances, models.PSPBalance{
			AccountReference: "spot",
			Asset:            bal.Asset,
			Amount:           totalAmount,
			CreatedAt:        time.Now(),
		})
	}

	return models.FetchNextBalancesResponse{
		Balances: balances,
		NewState: nil,
		HasMore:  false,
	}, nil
}

func parseBinanceAmount(amountStr string, asset string) (*big.Int, error) {
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

// parseDecimalString parses a decimal string to big.Int with a given precision
func parseDecimalString(s string, precision int) (*big.Int, error) {
	return parseAmountWithPrecision(s, precision)
}
