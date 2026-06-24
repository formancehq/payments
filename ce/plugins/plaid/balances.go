package plaid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/ce/plugins/plaid/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/plugins"
	"github.com/plaid/plaid-go/v34/plaid"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var pspAccount models.PSPAccount
	if err := json.Unmarshal(req.FromPayload, &pspAccount); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	pspBalance, err := toPSPBalance(pspAccount)
	if err != nil {
		if errors.Is(err, plugins.ErrCurrencyNotSupported) {
			// Skip unsupported currencies rather than failing: a retryable
			// error here would freeze balance ingestion for the account.
			p.logger.WithField("reference", pspAccount.Reference).Info("skipping balance with unsupported currency")
			return models.FetchNextBalancesResponse{}, nil
		}
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
	if !balance.Current.IsSet() || balance.Current.Get() == nil {
		return models.PSPBalance{}, fmt.Errorf("balance is not set")
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
		return models.PSPBalance{}, err
	}

	lastUpdated := balance.GetLastUpdatedDatetime()
	if lastUpdated.IsZero() {
		lastUpdated = time.Now()
	}

	return models.PSPBalance{
		AccountReference:        account.AccountId,
		CreatedAt:               lastUpdated.UTC(),
		Amount:                  amount,
		Asset:                   assetName,
		PsuID:                   pspAccount.PsuID,
		OpenBankingConnectionID: pspAccount.OpenBankingConnectionID,
	}, nil
}
