package routable

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/routable/mappers"
	"github.com/formancehq/payments/internal/models"
)

// Cap /v1/companies walks to once per 24h; see MAPPINGS.md §6.5.5.
const externalAccountsRefreshInterval = 24 * time.Hour

var now = time.Now

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	state, err := decodePageState(req.State)
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	if state.isStartOfCycle() && now().Sub(state.LastCompletedAt) < externalAccountsRefreshInterval {
		payload, err := json.Marshal(state)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, fmt.Errorf("marshaling state: %w", err)
		}
		return models.FetchNextExternalAccountsResponse{NewState: payload, HasMore: false}, nil
	}

	resp, err := p.client.ListCompanies(ctx, state.nextPage(), req.PageSize)
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, fmt.Errorf("listing companies (page=%d): %w", state.nextPage(), err)
	}

	accounts := make([]models.PSPAccount, 0, len(resp.Results))
	for _, co := range resp.Results {
		account, err := mappers.CompanyToPSPAccount(co)
		if err != nil {
			p.logger.Infof("skipping company %s: %v", co.ID, err)
			continue
		}
		accounts = append(accounts, account)
	}

	newState := pageState{Page: state.nextPage() + 1, LastCompletedAt: state.LastCompletedAt}
	if !resp.Links.HasMore() {
		newState.Page = 1
		newState.LastCompletedAt = now().UTC()
	}
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, fmt.Errorf("marshaling state: %w", err)
	}

	return models.FetchNextExternalAccountsResponse{
		ExternalAccounts: accounts,
		NewState:         payload,
		HasMore:          resp.Links.HasMore(),
	}, nil
}
