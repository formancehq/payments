package stripe

import (
	"context"
	"encoding/json"
	"github.com/stripe/stripe-go/v79"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, errors.New("missing from payload when fetching balances")
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	balance, err := p.client.GetAccountBalances(ctx, resolveAccount(from.Reference))
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	accountBalances := p.fromStripeBalanceToPSPBalances(balance, from.Reference)

	return models.FetchNextBalancesResponse{
		Balances: accountBalances,
		HasMore:  false,
	}, nil
}

func (p *Plugin) fromStripeBalanceToPSPBalances(from *stripe.Balance, accountReference string) []models.PSPBalance {
	var accountBalances []models.PSPBalance
	for _, available := range from.Available {
		timestamp := time.Now()
		accountBalances = append(accountBalances, models.PSPBalance{
			AccountReference: accountReference,
			Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, string(available.Currency)),
			Amount:           big.NewInt(available.Amount),
			CreatedAt:        timestamp,
		})
	}

	return accountBalances
}
