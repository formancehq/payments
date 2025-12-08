package tink

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var pspAccount models.PSPAccount
	if err := json.Unmarshal(req.FromPayload, &pspAccount); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	pspBalance, err := toPSPBalance(pspAccount)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	pspBalances := make([]models.PSPBalance, 0, 1)
	pspBalances = append(pspBalances, pspBalance)
	return models.FetchNextBalancesResponse{
		Balances: pspBalances,
	}, nil
}

func toPSPBalance(
	pspAccount models.PSPAccount,
) (models.PSPBalance, error) {

	var account client.Account
	if err := json.Unmarshal(pspAccount.Raw, &account); err != nil {
		return models.PSPBalance{}, err
	}

	balance := account.Balances.Booked

	amount, asset, err := MapTinkAmount(balance.Amount.Value.Value, balance.Amount.Value.Scale, balance.Amount.CurrencyCode)
	if err != nil {
		return models.PSPBalance{}, err
	}

	return models.PSPBalance{
		AccountReference:        account.ID,
		CreatedAt:               account.Dates.LastRefreshed.UTC(),
		Amount:                  amount,
		Asset:                   *asset,
		PsuID:                   pspAccount.PsuID,
		OpenBankingConnectionID: pspAccount.OpenBankingConnectionID, // Note -- currently Tink doesn't forward the connectionID, so this is mostly wishful thinking.
	}, nil
}
