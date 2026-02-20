package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req connector.FetchNextBalancesRequest) (connector.FetchNextBalancesResponse, error) {
	var from connector.PSPAccount
	if req.FromPayload == nil {
		return connector.FetchNextBalancesResponse{}, fmt.Errorf("missing from payload when fetching balances")
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	walletSymbol, err := walletSymbolFromAccount(from)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	response, err := p.client.GetBalancesForSymbol(ctx, walletSymbol, "", req.PageSize)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	now := time.Now().UTC()
	var balances []connector.PSPBalance
	for _, bal := range response.Balances {
		symbol := strings.ToUpper(strings.TrimSpace(bal.Symbol))

		precision, ok := supportedCurrenciesWithDecimal[symbol]
		if !ok {
			continue
		}

		amount, err := currency.GetAmountWithPrecisionFromString(bal.Amount, precision)
		if err != nil {
			return connector.FetchNextBalancesResponse{}, fmt.Errorf("failed to parse balance for %s: %w", symbol, err)
		}

		asset := currency.FormatAsset(supportedCurrenciesWithDecimal, symbol)

		balances = append(balances, connector.PSPBalance{
			AccountReference: from.Reference,
			Asset:            asset,
			Amount:           amount,
			CreatedAt:        now,
		})
	}

	return connector.FetchNextBalancesResponse{
		Balances: balances,
		HasMore:  false,
	}, nil
}

func walletSymbolFromAccount(from connector.PSPAccount) (string, error) {
	if from.DefaultAsset == nil {
		return "", fmt.Errorf("missing default asset in from payload")
	}

	symbol, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, *from.DefaultAsset)
	if err != nil {
		return "", fmt.Errorf("failed to parse default asset %q: %w", *from.DefaultAsset, err)
	}

	return symbol, nil
}
