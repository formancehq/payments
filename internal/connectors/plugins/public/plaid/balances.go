package plaid

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/plaid/plaid-go/v34/plaid"
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

	var account plaid.AccountBase
	if err := json.Unmarshal(pspAccount.Raw, &account); err != nil {
		return models.PSPBalance{}, err
	}

	balance := account.Balances
	if balance.Current.IsSet() == false {
		return models.PSPBalance{}, fmt.Errorf("balance is not set")
	}
	amountF := *balance.Current.Get()

	var curr string
	if balance.IsoCurrencyCode.IsSet() {
		curr = *balance.IsoCurrencyCode.Get()
	} else {
		curr = balance.GetUnofficialCurrencyCode()
	}

	amount, err := client.TranslatePlaidAmount(amountF, curr)
	if err != nil {
		return models.PSPBalance{}, err
	}
	precision, ok := currency.ISO4217Currencies[curr]
	if !ok {
		return models.PSPBalance{}, fmt.Errorf("unsupported currency: %s", curr)
	}
	asset := currency.FormatAssetWithPrecision(curr, precision)

	lastUpdated := balance.GetLastUpdatedDatetime()
	if lastUpdated.IsZero() {
		lastUpdated = time.Now()
	}

	return models.PSPBalance{
		AccountReference:        account.AccountId,
		CreatedAt:               lastUpdated.UTC(),
		Amount:                  amount,
		Asset:                   asset,
		PsuID:                   pspAccount.PsuID,
		OpenBankingConnectionID: pspAccount.OpenBankingConnectionID,
	}, nil
}
