package column

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

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

	balance, err := p.client.GetAccountBalances(ctx, from.Reference)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	var amount big.Int
	_, ok := amount.SetString(balance.AvailableAmount.String(), 10)
	if !ok {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to parse amount: %s", balance.AvailableAmount.String())
	}
	accountBalances := models.PSPBalance{
		AccountReference: from.Reference,
		Amount:           &amount,
		CreatedAt:        time.Now().UTC(),
	}
	if from.DefaultAsset != nil {
		accountBalances.Asset = *from.DefaultAsset
	}
	return models.FetchNextBalancesResponse{
		Balances: []models.PSPBalance{accountBalances},
		HasMore:  false,
	}, nil
}
