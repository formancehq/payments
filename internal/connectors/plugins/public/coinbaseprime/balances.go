package coinbaseprime

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/coinbase-samples/prime-sdk-go/model"
	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
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
	// Parse the amount string to big.Int
	amount, err := parseAmount(bal.Amount, bal.Symbol)
	if err != nil {
		return models.PSPBalance{}, fmt.Errorf("failed to parse amount: %w", err)
	}

	return models.PSPBalance{
		AccountReference: portfolioID,
		Asset:            bal.Symbol,
		Amount:           amount,
		CreatedAt:        time.Now(),
	}, nil
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
