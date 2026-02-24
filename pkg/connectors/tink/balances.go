package tink

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/pkg/connectors/tink/client"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req connector.FetchNextBalancesRequest) (connector.FetchNextBalancesResponse, error) {
	var pspAccount connector.PSPAccount
	if err := json.Unmarshal(req.FromPayload, &pspAccount); err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	pspBalance, err := toPSPBalance(pspAccount)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	pspBalances := make([]connector.PSPBalance, 0, 1)
	pspBalances = append(pspBalances, pspBalance)
	return connector.FetchNextBalancesResponse{
		Balances: pspBalances,
	}, nil
}

func toPSPBalance(
	pspAccount connector.PSPAccount,
) (connector.PSPBalance, error) {

	var account client.Account
	if err := json.Unmarshal(pspAccount.Raw, &account); err != nil {
		return connector.PSPBalance{}, err
	}

	balance := account.Balances.Booked

	amount, asset, err := MapTinkAmount(balance.Amount.Value.Value, balance.Amount.Value.Scale, balance.Amount.CurrencyCode)
	if err != nil {
		return connector.PSPBalance{}, err
	}

	return connector.PSPBalance{
		AccountReference:        account.ID,
		CreatedAt:               account.Dates.LastRefreshed.UTC(),
		Amount:                  amount,
		Asset:                   *asset,
		PsuID:                   pspAccount.PsuID,
		OpenBankingConnectionID: pspAccount.OpenBankingConnectionID, // Note -- currently Tink doesn't forward the connectionID, so this is mostly wishful thinking.
	}, nil
}
