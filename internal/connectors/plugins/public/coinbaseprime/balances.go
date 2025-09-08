package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	kind := from.Metadata["spec.coinbase.com/type"]
	var sdkBalances []*client.Balance
	var err error
	if kind == "wallet" {
		portfolioID := from.Metadata["spec.coinbase.com/portfolio_id"]
		sdkBalances, err = p.client.GetWalletBalance(ctx, portfolioID, from.Reference)
	} else {
		sdkBalances, err = p.client.GetAccountBalances(ctx, from.Reference)
	}
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	res := make([]models.PSPBalance, 0, len(sdkBalances))
	for _, b := range sdkBalances {
		symbol := strings.ToUpper(b.Symbol)
		precision, ok := supportedCurrenciesWithDecimal[symbol]
		if !ok {
			precision = 8
		}
		amount, err := currency.GetAmountWithPrecisionFromString(b.Amount, precision)
		if err != nil {
			return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to parse balance amount: %w", err)
		}
		asset := currency.FormatAsset(supportedCurrenciesWithDecimal, symbol)
		if asset == "" {
			asset = fmt.Sprintf("%s/%d", symbol, precision)
		}

		res = append(res, models.PSPBalance{
			AccountReference: from.Reference,
			CreatedAt:        time.Now().UTC(),
			Amount:           amount,
			Asset:            asset,
		})
	}

	return models.FetchNextBalancesResponse{
		Balances: res,
		HasMore:  false,
	}, nil
}
