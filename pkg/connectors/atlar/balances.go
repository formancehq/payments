package atlar

import (
	"context"
	"encoding/json"
	"math/big"

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

	account, err := p.client.GetV1AccountsID(ctx, from.Reference)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	// https://docs.atlar.com/docs/deprecations#v1-accountbalance
	// some accounts no longer return balance: we'll need to upgrade to API v2 to get those balances
	// but for now we can avoid a panic
	if account.Payload == nil || account.Payload.Balance == nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	balance := account.Payload.Balance
	balanceTimestamp, err := ParseAtlarTimestamp(balance.Timestamp)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	res := connector.PSPBalance{
		AccountReference: from.Reference,
		CreatedAt:        balanceTimestamp,
		Amount:           big.NewInt(*balance.Amount.Value),
		Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, *balance.Amount.Currency),
	}

	return connector.FetchNextBalancesResponse{
		Balances: []connector.PSPBalance{res},
		HasMore:  false,
	}, nil
}
