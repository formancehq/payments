package mangopay

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

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

	wallet, err := p.client.GetWallet(ctx, from.Reference)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	var amount big.Int
	_, ok := amount.SetString(wallet.Balance.Amount.String(), 10)
	if !ok {
		return connector.FetchNextBalancesResponse{}, fmt.Errorf("failed to parse amount: %s", wallet.Balance.Amount.String())
	}

	balance := connector.PSPBalance{
		AccountReference: from.Reference,
		CreatedAt:        time.Now().UTC(),
		Amount:           &amount,
		Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, wallet.Balance.Currency),
	}

	return connector.FetchNextBalancesResponse{
		Balances: []connector.PSPBalance{balance},
		HasMore:  false,
	}, nil
}
