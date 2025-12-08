package atlar

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/formancehq/go-libs/v3/currency"
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

	account, err := p.client.GetV1AccountsID(ctx, from.Reference)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	// https://docs.atlar.com/docs/deprecations#v1-accountbalance
	// some accounts no longer return balance: we'll need to upgrade to API v2 to get those balances
	// but for now we can avoid a panic
	if account.Payload == nil || account.Payload.Balance == nil {
		return models.FetchNextBalancesResponse{}, err
	}

	balance := account.Payload.Balance
	balanceTimestamp, err := ParseAtlarTimestamp(balance.Timestamp)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	res := models.PSPBalance{
		AccountReference: from.Reference,
		CreatedAt:        balanceTimestamp,
		Amount:           big.NewInt(*balance.Amount.Value),
		Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, *balance.Amount.Currency),
	}

	return models.FetchNextBalancesResponse{
		Balances: []models.PSPBalance{res},
		HasMore:  false,
	}, nil
}
