package teller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
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

	return models.FetchNextBalancesResponse{
		Balances: []models.PSPBalance{pspBalance},
	}, nil
}

func toPSPBalance(pspAccount models.PSPAccount) (models.PSPBalance, error) {
	var accountWithBalance tellerAccountWithBalance
	if err := json.Unmarshal(pspAccount.Raw, &accountWithBalance); err != nil {
		return models.PSPBalance{}, err
	}

	if accountWithBalance.Balance == nil {
		return models.PSPBalance{}, fmt.Errorf("balance is not available for account %s", accountWithBalance.Account.ID)
	}

	balance := accountWithBalance.Balance

	// Prefer available balance over ledger
	amountStr := balance.Available
	if amountStr == "" {
		amountStr = balance.Ledger
	}
	if amountStr == "" {
		return models.PSPBalance{}, fmt.Errorf("no balance value available for account %s", accountWithBalance.Account.ID)
	}

	curr := strings.ToUpper(accountWithBalance.Account.Currency)

	precision, err := currency.GetPrecision(supportedCurrenciesWithDecimal, curr)
	if err != nil {
		return models.PSPBalance{}, err
	}

	amount, err := currency.GetAmountWithPrecisionFromString(amountStr, precision)
	if err != nil {
		return models.PSPBalance{}, err
	}

	return models.PSPBalance{
		AccountReference:        accountWithBalance.Account.ID,
		CreatedAt:               time.Now().UTC(),
		Amount:                  amount,
		Asset:                   currency.FormatAssetWithPrecision(curr, precision),
		PsuID:                   pspAccount.PsuID,
		OpenBankingConnectionID: pspAccount.OpenBankingConnectionID,
	}, nil
}
