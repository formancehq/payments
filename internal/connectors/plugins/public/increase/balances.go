package increase

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
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

	balance, atTime, err := p.client.GetAccountBalance(ctx, from.Reference)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	var accountBalances []models.PSPBalance
	precision, err := currency.GetPrecision(supportedCurrenciesWithDecimal, *from.DefaultAsset)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	amount, err := currency.GetAmountWithPrecisionFromString(balance.AvailableBalance.String(), precision)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	asset := currency.FormatAsset(supportedCurrenciesWithDecimal, *from.DefaultAsset)

	accountBalances = append(accountBalances, models.PSPBalance{
		AccountReference: balance.AccountID,
		Amount:           amount,
		Asset:            asset,
		CreatedAt:        atTime,
	})

	return models.FetchNextBalancesResponse{
		Balances: accountBalances,
		HasMore:  false,
	}, nil
}
