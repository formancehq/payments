package bankingcircle

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/bankingcircle/client"
	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	LastAccountID   string    `json:"lastAccountID"`
	FromOpeningDate time.Time `json:"fromOpeningDate"`
}

func (p Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	newState := accountsState{
		LastAccountID:   oldState.LastAccountID,
		FromOpeningDate: oldState.FromOpeningDate,
	}

	var accounts []models.PSPAccount
	hasMore := false
	for page := 1; ; page++ {
		pagedAccounts, err := p.client.GetAccounts(ctx, page, req.PageSize, oldState.FromOpeningDate)
		if err != nil {
			return models.FetchNextAccountsResponse{}, nil
		}

		if len(pagedAccounts) == 0 {
			hasMore = false
			break
		}

		filteredAccounts := filterAccounts(pagedAccounts, oldState.LastAccountID)
		accounts, err = fillAccounts(accounts, filteredAccounts)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		needMore := true
		needMore, hasMore, accounts = shouldFetchMore(accounts, req.PageSize)
		if !needMore {
			break
		}
	}

	if len(accounts) > 0 {
		newState.LastAccountID = accounts[len(accounts)-1].Reference
		newState.FromOpeningDate = accounts[len(accounts)-1].CreatedAt
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func filterAccounts(pagedAccounts []client.Account, lastAccountID string) []client.Account {
	if lastAccountID == "" {
		return pagedAccounts
	}

	var filteredAccounts []client.Account
	found := false
	for _, account := range pagedAccounts {
		if !found && account.AccountID != lastAccountID {
			continue
		}

		if !found && account.AccountID == lastAccountID {
			found = true
			continue
		}

		filteredAccounts = append(filteredAccounts, account)
	}

	if !found {
		return pagedAccounts
	}

	return filteredAccounts
}

func fillAccounts(
	accounts []models.PSPAccount,
	pagedAccounts []client.Account,
) ([]models.PSPAccount, error) {
	for _, account := range pagedAccounts {
		openingDate, err := time.Parse("2006-01-02T15:04:05.999999999+00:00", account.OpeningDate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse opening date: %w", err)
		}

		raw, err := json.Marshal(account)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal account: %w", err)
		}

		accounts = append(accounts, models.PSPAccount{
			Reference:    account.AccountID,
			CreatedAt:    openingDate,
			Name:         &account.AccountDescription,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, account.Currency)),
			Raw:          raw,
		})
	}

	return accounts, nil
}
