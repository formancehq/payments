package column

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

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

	balance, err := p.client.GetAccountBalances(ctx, from.Reference)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	var amount big.Int
	_, ok := amount.SetString(balance.AvailableAmount.String(), 10)
	if !ok {
		return connector.FetchNextBalancesResponse{}, fmt.Errorf("failed to parse amount: %s", balance.AvailableAmount.String())
	}
	accountBalances := connector.PSPBalance{
		AccountReference: from.Reference,
		Amount:           &amount,
		CreatedAt:        time.Now().UTC(),
	}
	if from.DefaultAsset != nil {
		accountBalances.Asset = *from.DefaultAsset
	}
	return connector.FetchNextBalancesResponse{
		Balances: []connector.PSPBalance{accountBalances},
		HasMore:  false,
	}, nil
}
