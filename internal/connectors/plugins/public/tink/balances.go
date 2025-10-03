package tink

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/formancehq/go-libs/v3/currency"
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

	amount := balance.Amount.Value.Value
	amountBigInt, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return models.PSPBalance{}, fmt.Errorf("failed to parse amount: %s", amount)
	}
	precision, ok := currency.ISO4217Currencies[balance.Amount.CurrencyCode]
	if !ok {
		return models.PSPBalance{}, fmt.Errorf("unsupported currency: %s", balance.Amount.CurrencyCode)
	}
	asset := currency.FormatAssetWithPrecision(balance.Amount.CurrencyCode, precision)

	return models.PSPBalance{
		AccountReference: account.ID,
		CreatedAt:        account.Dates.LastRefreshed.UTC(),
		Amount:           amountBigInt,
		Asset:            asset,
	}, nil
}
