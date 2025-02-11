package increase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	NextCursor string `json:"next_cursor"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	accounts := make([]models.PSPAccount, 0, req.PageSize)
	hasMore := false
	pagedAccounts, nextCursor, err := p.client.GetAccounts(ctx, req.PageSize, oldState.NextCursor)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accounts, err = fillAccounts(pagedAccounts, accounts, req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	hasMore = nextCursor != ""

	newState := accountsState{
		NextCursor: nextCursor,
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

func fillAccounts(
	pagedAccounts []*client.Account,
	accounts []models.PSPAccount,
	pageSize int,
) ([]models.PSPAccount, error) {
	for _, account := range pagedAccounts {
		if len(accounts) >= pageSize {
			break
		}

		createdTime, err := time.Parse("2006-01-02T15:04:05.999-0700", account.CreatedAt)
		if err != nil {
			return nil, err
		}

		raw, err := json.Marshal(account)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, models.PSPAccount{
			Reference:    account.ID,
			CreatedAt:    createdTime,
			Name:         &account.Name,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, account.Currency)),
			Raw:          raw,
			Metadata: map[string]string{
				"type":     account.Type,
				"bank":     account.Bank,
				"currency": account.Currency,
				"status":   account.Status,
			},
		})
	}

	return accounts, nil
}
