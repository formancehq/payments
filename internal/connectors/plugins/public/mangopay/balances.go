package mangopay

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

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

	wallet, err := p.client.GetWallet(ctx, from.Reference)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	var amount big.Int
	_, ok := amount.SetString(wallet.Balance.Amount.String(), 10)
	if !ok {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to parse amount: %s", wallet.Balance.Amount.String())
	}

	balance := models.PSPBalance{
		AccountReference: from.Reference,
		CreatedAt:        time.Now().UTC(),
		Amount:           &amount,
		Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, wallet.Balance.Currency),
	}

	return models.FetchNextBalancesResponse{
		Balances: []models.PSPBalance{balance},
		HasMore:  false,
	}, nil
}
