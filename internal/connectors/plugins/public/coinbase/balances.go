package coinbase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("missing from payload when fetching balances")
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	walletSymbol, err := walletSymbolFromAccount(from)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	response, err := p.client.GetBalancesForSymbol(ctx, walletSymbol, "", req.PageSize)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	now := time.Now().UTC()
	var balances []models.PSPBalance
	for _, bal := range response.Balances {
		symbol := strings.ToUpper(strings.TrimSpace(bal.Symbol))

		precision, ok := supportedCurrenciesWithDecimal[symbol]
		if !ok {
			continue
		}

		amount, err := currency.GetAmountWithPrecisionFromString(bal.Amount, precision)
		if err != nil {
			return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to parse balance for %s: %w", symbol, err)
		}

		asset := currency.FormatAsset(supportedCurrenciesWithDecimal, symbol)

		balances = append(balances, models.PSPBalance{
			AccountReference: from.Reference,
			Asset:            asset,
			Amount:           amount,
			CreatedAt:        now,
		})
	}

	return models.FetchNextBalancesResponse{
		Balances: balances,
		HasMore:  false,
	}, nil
}

func walletSymbolFromAccount(from models.PSPAccount) (string, error) {
	if symbol := strings.TrimSpace(from.Metadata["symbol"]); symbol != "" {
		return strings.ToUpper(symbol), nil
	}

	if len(from.Raw) > 0 {
		var wallet struct {
			Symbol string `json:"symbol"`
		}

		if err := json.Unmarshal(from.Raw, &wallet); err != nil {
			return "", fmt.Errorf("failed to parse wallet raw payload: %w", err)
		}

		if symbol := strings.TrimSpace(wallet.Symbol); symbol != "" {
			return strings.ToUpper(symbol), nil
		}
	}

	return "", fmt.Errorf("missing wallet symbol in from payload")
}
