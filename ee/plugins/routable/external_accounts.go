package routable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/models"
)

// fetchNextExternalAccounts pages through Routable companies and emits each
// one as an EXTERNAL PSPAccount. Unlike the Generic-Connector adapter we no
// longer fan out to GET /v1/companies/{id}/payment-methods per row: the
// expensive N+1 is deferred until payable creation, where the resolved
// payment method actually matters.
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
		account, err := p.companyToPSPAccount(co)
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

func (p *Plugin) companyToPSPAccount(co client.Company) (models.PSPAccount, error) {
	raw, err := json.Marshal(co)
	if err != nil {
		return models.PSPAccount{}, fmt.Errorf("marshaling raw: %w", err)
	}
	displayName := co.DisplayName
	if displayName == "" {
		displayName = co.BusinessName
	}
	return models.PSPAccount{
		Reference: co.ID,
		CreatedAt: co.CreatedAt,
		Name:      pointerOrNil(displayName),
		Metadata:  companyMetadata(co),
		Raw:       raw,
	}, nil
}
