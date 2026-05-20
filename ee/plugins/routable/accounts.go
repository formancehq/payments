package routable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/routable/mappers"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	state, err := decodePageState(req.State)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	resp, err := p.client.ListAccounts(ctx, state.nextPage(), req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("listing accounts (page=%d): %w", state.nextPage(), err)
	}

	accounts := make([]models.PSPAccount, 0, len(resp.Results))
	for _, a := range resp.Results {
		account, err := mappers.SettingsAccountToPSPAccount(a)
		if err != nil {
			p.logger.Infof("skipping settings account %s: %v", a.ID, err)
			continue
		}
		accounts = append(accounts, account)
	}

	newState := pageState{Page: state.nextPage() + 1}
	if !resp.Links.HasMore() {
		newState.Page = 1 // restart on next cycle
	}
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("marshaling state: %w", err)
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  resp.Links.HasMore(),
	}, nil
}

func decodePageState(raw json.RawMessage) (pageState, error) {
	var s pageState
	if len(raw) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, fmt.Errorf("decoding state: %w", err)
	}
	return s, nil
}
