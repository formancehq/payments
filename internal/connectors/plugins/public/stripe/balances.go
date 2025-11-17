package stripe

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v80"
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

	var accountBalances []models.PSPBalance
	for _, available := range balance.Available {
		timestamp := time.Now()
		accountBalances = append(accountBalances, toPSPBalance(from.Reference, timestamp, available))
	}

	return models.FetchNextBalancesResponse{
		Balances: accountBalances,
		HasMore:  false,
	}, nil
}

func toPSPBalance(accountRef string, createdAt time.Time, a *stripe.Amount) models.PSPBalance {
	return models.PSPBalance{
		AccountReference: accountRef,
		Amount:           big.NewInt(a.Amount),
		Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, string(a.Currency)),
		CreatedAt:        createdAt,
	}
}
