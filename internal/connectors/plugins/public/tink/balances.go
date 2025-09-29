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
	var from models.OpenBankingForwardedUserFromPayload
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	var webhook fetchNextDataRequest
	if err := json.Unmarshal(from.FromPayload, &webhook); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	balance, err := p.client.GetAccountBalances(ctx, webhook.ExternalUserID, webhook.AccountID)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	pspBalance, err := toPSPBalance(balance, webhook.AccountID)
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
	balance client.AccountBalanceResponse,
	accountReference string,
) (models.PSPBalance, error) {

	amount := new(big.Int)
	amount, ok := amount.SetString(balance.Balances.Booked.ValueInMinorUnit.String(), 10)
	if !ok {
		return models.PSPBalance{}, fmt.Errorf("failed to parse amount: %s", balance.Balances.Booked.ValueInMinorUnit.String())
	}

	precision, ok := currency.ISO4217Currencies[balance.Balances.Booked.CurrencyCode]
	if !ok {
		return models.PSPBalance{}, fmt.Errorf("unsupported currency: %s", balance.Balances.Booked.CurrencyCode)
	}
	asset := currency.FormatAssetWithPrecision(balance.Balances.Booked.CurrencyCode, precision)

	return models.PSPBalance{
		AccountReference: accountReference,
		CreatedAt:        balance.Refreshed.UTC(),
		Amount:           amount,
		Asset:            asset,
	}, nil
}
