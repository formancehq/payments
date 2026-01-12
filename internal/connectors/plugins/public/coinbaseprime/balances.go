package coinbaseprime

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/coinbase-samples/prime-sdk-go/model"
	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/assets"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	// Fetch balances for the portfolio
	balancesResp, err := p.client.GetPortfolioBalances(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to get portfolio balances: %w", err)
	}

	balances := make([]models.PSPBalance, 0, len(balancesResp.Balances))
	for _, bal := range balancesResp.Balances {
		balance, err := modelBalanceToBalance(bal, p.config.PortfolioID)
		if err != nil {
			p.logger.Errorf("failed to convert balance for %s: %v", bal.Symbol, err)
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

func modelBalanceToBalance(bal *model.Balance, portfolioID string) (models.PSPBalance, error) {
	// Normalize the symbol to match asset validation requirements
	normalizedSymbol, err := normalizeSymbol(bal.Symbol)
	if err != nil {
		return models.PSPBalance{}, fmt.Errorf("invalid symbol %q: %w", bal.Symbol, err)
	}

	// Parse the amount string to big.Int
	amount, err := parseAmount(bal.Amount, normalizedSymbol)
	if err != nil {
		return models.PSPBalance{}, fmt.Errorf("failed to parse amount: %w", err)
	}

	return models.PSPBalance{
		AccountReference: portfolioID,
		Asset:            normalizedSymbol,
		Amount:           amount,
		CreatedAt:        time.Now(),
	}, nil
}

// normalizeSymbol converts a Coinbase Prime symbol to the standard format
// required by the asset validation (uppercase, alphanumeric only).
func normalizeSymbol(symbol string) (string, error) {
	if symbol == "" {
		return "", fmt.Errorf("empty symbol")
	}

	// Convert to uppercase
	normalized := strings.ToUpper(strings.TrimSpace(symbol))

	// Validate the normalized symbol
	if !assets.IsValid(normalized) {
		return "", fmt.Errorf("symbol %q does not match required format", normalized)
	}

	return normalized, nil
}

func parseAmount(amountStr string, symbol string) (*big.Int, error) {
	if amountStr == "" {
		return big.NewInt(0), nil
	}

	// Get precision for the currency
	precision := GetPrecision(symbol)

	// Parse as decimal and convert to integer with precision
	amount, err := currency.GetAmountWithPrecisionFromString(amountStr, precision)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount %s: %w", amountStr, err)
	}

	return amount, nil
}
