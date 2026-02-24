package fireblocks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/pkg/connector"
)

type accountsState struct {
	NextCursor string `json:"nextCursor"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req connector.FetchNextAccountsRequest) (connector.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}
	}

	resp, err := p.client.GetVaultAccountsPaged(ctx, oldState.NextCursor, int(req.PageSize))
	if err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	accounts := make([]connector.PSPAccount, 0, len(resp.Accounts))
	for _, account := range resp.Accounts {
		raw, err := json.Marshal(account)
		if err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}

		createdAt := time.Now()
		if account.CreationDate > 0 {
			createdAt = time.UnixMilli(account.CreationDate)
		}

		accounts = append(accounts, connector.PSPAccount{
			Reference: account.ID,
			CreatedAt: createdAt,
			Name:      &account.Name,
			Raw:       raw,
		})
	}

	newState := accountsState{
		NextCursor: resp.Paging.After,
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	hasMore := resp.Paging.After != ""

	return connector.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}
