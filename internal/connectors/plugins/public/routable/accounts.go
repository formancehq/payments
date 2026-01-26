package routable

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type accountsState struct{}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	accounts := make([]models.PSPAccount, 0, 1)
	hasMore := false

	pagedAccounts, err := p.client.GetAccounts(ctx, 0, 1)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	if len(pagedAccounts) > 0 {
		createdAt, _ := time.Parse(time.RFC3339, pagedAccounts[0].CreatedAt)
		raw, _ := json.Marshal(pagedAccounts[0])
		acc := models.PSPAccount{
			Reference: pagedAccounts[0].ID,
			CreatedAt: createdAt,
			Name:      &pagedAccounts[0].Name,
			Metadata:  map[string]string{"spec.formance.com/generic_provider": ProviderName},
			Raw:       raw,
		}
		accounts = append(accounts, acc)
	}

	_, hasMore = pagination.ShouldFetchMore(accounts, pagedAccounts, req.PageSize)

	payload, _ := json.Marshal(accountsState{})
	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}
