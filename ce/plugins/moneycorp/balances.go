package moneycorp

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/payments/pkg/domain/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	balances, err := p.client.GetAccountBalances(ctx, from.Reference)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	var accountBalances []models.PSPBalance
	for _, balance := range balances {
		precision, err := currency.GetPrecision(supportedCurrenciesWithDecimal, balance.Attributes.CurrencyCode)
		if err != nil {
			if errors.Is(err, currency.ErrMissingCurrencies) {
				// Skip unsupported currencies rather than failing: a retryable
				// error here would freeze balance ingestion for the account.
				p.logger.WithField("currency", balance.Attributes.CurrencyCode).Info("skipping balance with unsupported currency")
				continue
			}
			return models.FetchNextBalancesResponse{}, err
		}

		amount, err := currency.GetAmountWithPrecisionFromString(balance.Attributes.AvailableBalance.String(), precision)
		if err != nil {
			return models.FetchNextBalancesResponse{}, err
		}

		accountBalances = append(accountBalances, models.PSPBalance{
			AccountReference: from.Reference,
			CreatedAt:        time.Now(),
			Amount:           amount,
			Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, balance.Attributes.CurrencyCode),
		})
	}

	return models.FetchNextBalancesResponse{
		Balances: accountBalances,
		HasMore:  false,
	}, nil
}
