package plaid

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/pkg/connectors/plaid/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/plaid/plaid-go/v34/plaid"
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

	var account plaid.AccountBase
	if err := json.Unmarshal(pspAccount.Raw, &account); err != nil {
		return connector.PSPBalance{}, err
	}

	balance := account.Balances
	if !balance.Current.IsSet() || balance.Current.Get() == nil {
		return connector.PSPBalance{}, fmt.Errorf("balance is not set")
	}
	amountF := *balance.Current.Get()

	var curr string
	if balance.IsoCurrencyCode.IsSet() && balance.IsoCurrencyCode.Get() != nil {
		curr = *balance.IsoCurrencyCode.Get()
	} else {
		curr = balance.GetUnofficialCurrencyCode()
	}

	amount, assetName, err := client.TranslatePlaidAmount(amountF, curr)
	if err != nil {
		return connector.PSPBalance{}, err
	}

	lastUpdated := balance.GetLastUpdatedDatetime()
	if lastUpdated.IsZero() {
		lastUpdated = time.Now()
	}

	return connector.PSPBalance{
		AccountReference:        account.AccountId,
		CreatedAt:               lastUpdated.UTC(),
		Amount:                  amount,
		Asset:                   assetName,
		PsuID:                   pspAccount.PsuID,
		OpenBankingConnectionID: pspAccount.OpenBankingConnectionID,
	}, nil
}
