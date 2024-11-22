package atlar

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
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
