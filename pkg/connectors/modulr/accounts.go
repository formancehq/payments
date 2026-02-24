package modulr

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/modulr/client"
	"github.com/formancehq/payments/pkg/connector"
)

type accountsState struct {
	LastCreatedAt time.Time `json:"lastCreatedAt"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req connector.FetchNextAccountsRequest) (connector.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}
	}

	newState := accountsState{
		LastCreatedAt: oldState.LastCreatedAt,
	}

	var accounts []connector.PSPAccount
	needMore := false
	hasMore := false
	for page := 0; ; page++ {
		pagedAccounts, err := p.client.GetAccounts(ctx, page, req.PageSize, oldState.LastCreatedAt)
		if err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}

		accounts, err = fillAccounts(pagedAccounts, accounts, oldState, req.PageSize)
		if err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}

		needMore, hasMore = connector.ShouldFetchMore(accounts, pagedAccounts, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	if !needMore {
		accounts = accounts[:req.PageSize]
	}

	if len(accounts) > 0 {
		newState.LastCreatedAt = accounts[len(accounts)-1].CreatedAt
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	return connector.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func fillAccounts(
	pagedAccounts []client.Account,
	accounts []connector.PSPAccount,
	oldState accountsState,
	pageSize int,
) ([]connector.PSPAccount, error) {
	for _, account := range pagedAccounts {
		if len(accounts) >= pageSize {
			break
		}

		createdTime, err := time.Parse("2006-01-02T15:04:05.999-0700", account.CreatedDate)
		if err != nil {
			return nil, err
		}

		switch createdTime.Compare(oldState.LastCreatedAt) {
		case -1, 0:
			// Account already ingested, skip
			continue
		default:
		}

		raw, err := json.Marshal(account)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, connector.PSPAccount{
			Reference:    account.ID,
			CreatedAt:    createdTime,
			Name:         &account.Name,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, account.Currency)),
			Raw:          raw,
		})
	}

	return accounts, nil
}
