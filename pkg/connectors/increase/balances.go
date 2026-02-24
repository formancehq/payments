package increase

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

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

	balance, atTime, err := p.client.GetAccountBalance(ctx, from.Reference)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	var amount big.Int
	_, ok := amount.SetString(balance.AvailableBalance.String(), 10)
	if !ok {
		return connector.FetchNextBalancesResponse{}, fmt.Errorf("failed to parse amount: %s", balance.AvailableBalance.String())
	}

	accountBalances := connector.PSPBalance{
		AccountReference: balance.AccountID,
		Amount:           &amount,
		CreatedAt:        atTime,
	}
	if from.DefaultAsset != nil {
		accountBalances.Asset = *from.DefaultAsset
	}

	return connector.FetchNextBalancesResponse{
		Balances: []connector.PSPBalance{accountBalances},
		HasMore:  false,
	}, nil
}
