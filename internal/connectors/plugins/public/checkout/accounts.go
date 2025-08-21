package checkout

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	LastPage int `json:"lastPage"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	const page = 0

	pagedAccounts, err := p.client.GetAccounts(ctx, page, req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accounts := make([]models.PSPAccount, 0, len(pagedAccounts))
	for _, acc := range pagedAccounts {
		raw, _ := json.Marshal(acc)

		md := map[string]string{
			"status": acc.Status,
		}

		var namePtr *string
		if acc.Name != "" {
			n := acc.Name
			namePtr = &n
		}

		accounts = append(accounts, models.PSPAccount{
			Reference:    acc.ID,
			Name:         namePtr,
			DefaultAsset: nil,
			Metadata:     md,
			Raw:          raw,
			CreatedAt:    time.Now().UTC(),
		})
	}

	if req.PageSize > 0 && len(accounts) > req.PageSize {
		accounts = accounts[:req.PageSize]
	}

	newState, _ := json.Marshal(accountsState{LastPage: 0})

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: newState,
		HasMore:  false,
	}, nil
}
