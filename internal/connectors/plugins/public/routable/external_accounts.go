package routable

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
)

type externalAccountsState struct {
	NextPage int `json:"nextPage"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	state := externalAccountsState{}
	if len(req.State) != 0 {
		_ = json.Unmarshal(req.State, &state)
	}
	page := state.NextPage
	if page == 0 {
		page = 1
	}

	companies, err := p.client.GetExternalAccounts(ctx, page, req.PageSize)
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	accounts := make([]models.PSPAccount, 0, len(companies))
	for _, c := range companies {
		createdAt, _ := time.Parse(time.RFC3339, c.CreatedAt)
		raw, _ := json.Marshal(c)
		name := c.DisplayName
		acc := models.PSPAccount{
			Reference:    c.ID,
			CreatedAt:    createdAt,
			Name:         &name,
			Metadata:     map[string]string{"spec.formance.com/generic_provider": ProviderName},
			Raw:          raw,
			DefaultAsset: nil,
		}
		accounts = append(accounts, acc)
	}

	hasMore := len(companies) == req.PageSize
	newState := externalAccountsState{NextPage: page + 1}
	payload, _ := json.Marshal(newState)
	return models.FetchNextExternalAccountsResponse{
		ExternalAccounts: accounts,
		NewState:         payload,
		HasMore:          hasMore,
	}, nil
}
