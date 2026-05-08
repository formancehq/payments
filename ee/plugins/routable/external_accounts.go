package routable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/routable/mappers"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	state, err := decodePageState(req.State)
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
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

	newState := pageState{Page: state.nextPage() + 1}
	if !resp.Links.HasMore() {
		newState.Page = 1
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
