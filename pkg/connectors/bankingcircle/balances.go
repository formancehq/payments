package bankingcircle

import (
	"context"
	"encoding/json"
	"math/big"
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

	account, err := p.client.GetAccount(ctx, from.Reference)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	var balances []connector.PSPBalance
	for _, balance := range account.Balances {
		// Note(polo): the last transaction timestamp is wrong in the banking
		// circle response. We will use the current time instead.
		lastTransactionTimestamp := time.Now().UTC()

		precision := supportedCurrenciesWithDecimal[balance.Currency]

		beginOfDayAmount, err := currency.GetAmountWithPrecisionFromString(balance.BeginOfDayAmount.String(), precision)
		if err != nil {
			return connector.FetchNextBalancesResponse{}, err
		}

		intraDayAmount, err := currency.GetAmountWithPrecisionFromString(balance.IntraDayAmount.String(), precision)
		if err != nil {
			return connector.FetchNextBalancesResponse{}, err
		}

		amount := big.NewInt(0).Add(beginOfDayAmount, intraDayAmount)

		balances = append(balances, connector.PSPBalance{
			AccountReference: from.Reference,
			CreatedAt:        lastTransactionTimestamp,
			Amount:           amount,
			Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, balance.Currency),
		})
	}

	return connector.FetchNextBalancesResponse{
		Balances: balances,
		HasMore:  false,
	}, nil
}
