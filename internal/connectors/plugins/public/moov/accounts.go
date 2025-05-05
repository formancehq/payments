package moov

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
	"github.com/moovfinancial/moov-go/pkg/moov"
)

type accountsState struct {
	Skip int `json:"skip"`
}

func (p *Plugin) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	if req.Name != "accounts" {
		return models.FetchNextOthersResponse{}, plugins.ErrNotImplemented
	}

	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextOthersResponse{}, err
		}
	}

	newState := accountsState{
		Skip: oldState.Skip,
	}

	others := make([]models.PSPOther, 0, req.PageSize)
	needMore := false
	hasMore := false

	accounts, hasMoreAccounts, err := p.client.GetAccounts(ctx, oldState.Skip, req.PageSize)
	if err != nil {
		return models.FetchNextOthersResponse{}, err
	}

	for _, account := range accounts {
		raw, err := json.Marshal(account)
		if err != nil {
			return models.FetchNextOthersResponse{}, err
		}

		others = append(others, models.PSPOther{
			ID:    account.ID,
			Other: raw,
		})
	}

	needMore, hasMore = pagination.ShouldFetchMore(others, accounts, req.PageSize)
	if !needMore {
		others = others[:req.PageSize]
	}

	// Update state for next fetch
	newState.Skip += len(accounts)

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextOthersResponse{}, err
	}

	return models.FetchNextOthersResponse{
		Others:   others,
		NewState: payload,
		HasMore:  hasMore || hasMoreAccounts,
	}, nil
}