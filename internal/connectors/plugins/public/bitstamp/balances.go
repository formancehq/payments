package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

// fetchNextBalances retrieves all currency balances for a specific Bitstamp account.
// The method:
// 1. Identifies the account from the request payload
// 2. Calls Bitstamp's account_balances API endpoint
// 3. Converts balance amounts using appropriate currency precision (fiat vs crypto)
// 4. Returns all balances in a single response (no pagination)
func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}
	
	p.logger.Infof("fetchNextBalances called with account reference: %s", from.Reference)

	// Find the account by reference
	var targetAccount *client.Account
	for _, account := range p.client.GetAllAccounts() {
		if account.ID == from.Reference {
			targetAccount = account
			break
		}
	}
	if targetAccount == nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("account not found: %s", from.Reference)
	}

	// Fetch balances for this specific account
	p.logger.Infof("Fetching balances for account reference: %s (ID: %s, Name: %s)", from.Reference, targetAccount.ID, targetAccount.Name)
	balances, err := p.client.GetAccountBalances(ctx, targetAccount)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	p.logger.Infof("Retrieved %d balances from Bitstamp for account %s", len(balances), from.Reference)

	accountBalances := make([]models.PSPBalance, 0, len(balances))
	for _, balance := range balances {
		symbol := strings.ToUpper(balance.Currency)
		precision, ok := supportedCurrenciesWithDecimal[symbol]
		if !ok {
			precision = 8
		}

		amount, err := currency.GetAmountWithPrecisionFromString(balance.Total, precision)
		if err != nil {
			return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to parse balance amount: %w", err)
		}
		asset := currency.FormatAsset(supportedCurrenciesWithDecimal, symbol)
		if asset == "" {
			asset = fmt.Sprintf("%s/%d", symbol, precision)
		}

		pspBalance := models.PSPBalance{
			AccountReference: from.Reference,
			CreatedAt:        time.Now().UTC(),
			Amount:           amount,
			Asset:            asset,
		}
		p.logger.Infof("Created PSPBalance: account=%s, asset=%s, amount=%s", pspBalance.AccountReference, pspBalance.Asset, amount.String())
		accountBalances = append(accountBalances, pspBalance)
	}

	p.logger.Infof("Returning %d PSPBalances for storage", len(accountBalances))

	return models.FetchNextBalancesResponse{
		Balances: accountBalances,
		HasMore:  false,
	}, nil
}
