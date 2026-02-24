package moneycorp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req connector.FetchNextBalancesRequest) (connector.FetchNextBalancesResponse, error) {
	var from connector.PSPAccount
	if req.FromPayload == nil {
		return connector.FetchNextBalancesResponse{}, connector.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	balances, err := p.client.GetAccountBalances(ctx, from.Reference)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	var accountBalances []connector.PSPBalance
	for _, balance := range balances {
		precision, err := currency.GetPrecision(supportedCurrenciesWithDecimal, balance.Attributes.CurrencyCode)
		if err != nil {
			return connector.FetchNextBalancesResponse{}, err
		}

		amount, err := currency.GetAmountWithPrecisionFromString(balance.Attributes.AvailableBalance.String(), precision)
		if err != nil {
			return connector.FetchNextBalancesResponse{}, err
		}

		accountBalances = append(accountBalances, connector.PSPBalance{
			AccountReference: from.Reference,
			CreatedAt:        time.Now(),
			Amount:           amount,
			Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, balance.Attributes.CurrencyCode),
		})
	}

	return connector.FetchNextBalancesResponse{
		Balances: accountBalances,
		HasMore:  false,
	}, nil
}
