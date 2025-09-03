package coinbaseprime

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	// TODO: if needed, uncomment the following lines to get the related account in request
	// var from models.PSPAccount
	// if req.FromPayload == nil {
	// 	return models.FetchNextBalancesResponse{}, models.ErrMissingFromPayloadInRequest
	// }
	// if err := json.Unmarshal(req.FromPayload, &from); err != nil {
	// 	return models.FetchNextBalancesResponse{}, err
	// }

	balances, err := p.client.GetAccountBalances(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	var accountBalances []models.PSPBalance
	for _, balance := range balances {
		// TODO: You can use the following method to extract the amount and the
		// asset from the PSP data if you're not already in minor units currency.
		// precision, err := currency.GetPrecision(supportedCurrenciesWithDecimal, balance.Attributes.CurrencyCode)
		// if err != nil {
		// 	return models.FetchNextBalancesResponse{}, err
		// }

		// amount, err := currency.GetAmountWithPrecisionFromString(balance.Attributes.AvailableBalance.String(), precision)
		// if err != nil {
		// 	return models.FetchNextBalancesResponse{}, err
		// }

		// asset := currency.FormatAsset(supportedCurrenciesWithDecimal, balance.Attributes.CurrencyCode)

		// TODO: translate PSP balance to formance balance object
		_ = balance
		accountBalances = append(accountBalances, models.PSPBalance{})
	}

	return models.FetchNextBalancesResponse{
		Balances: accountBalances,
		HasMore:  false,
	}, nil
}
