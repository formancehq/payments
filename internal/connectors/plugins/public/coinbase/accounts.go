package coinbase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	Fetched bool `json:"fetched"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	// Coinbase Exchange returns all accounts in a single call (no pagination)
	// We only fetch once per polling cycle
	if oldState.Fetched {
		return models.FetchNextAccountsResponse{
			HasMore: false,
		}, nil
	}

	rawAccounts, err := p.client.GetAccounts(ctx)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accounts := make([]models.PSPAccount, 0, len(rawAccounts))
	for _, acc := range rawAccounts {
		_, ok := supportedCurrenciesWithDecimal[acc.Currency]
		if !ok {
			// Skip accounts with unsupported currencies
			continue
		}

		raw, err := json.Marshal(acc)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		name := acc.Currency + " Wallet"
		defaultAsset := currency.FormatAsset(supportedCurrenciesWithDecimal, acc.Currency)

		accounts = append(accounts, models.PSPAccount{
			Reference: acc.ID,
			// Note: Coinbase Exchange API doesn't provide account creation date,
			// so we use the current time as a fallback
			CreatedAt:    time.Now().UTC(),
			Name:         &name,
			DefaultAsset: &defaultAsset,
			Metadata: map[string]string{
				"currency":        acc.Currency,
				"profile_id":      acc.ProfileID,
				"trading_enabled": boolToString(acc.TradingEnabled),
			},
			Raw: raw,
		})
	}

	newState := accountsState{Fetched: true}
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  false,
	}, nil
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
