package bitstamp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
)

// accountsState tracks pagination state for fetching accounts.
// Bitstamp returns accounts from configuration rather than paginated API,
// so this remains empty but is kept for consistency with the framework.
type accountsState struct {
	// TODO: accountsState will be used to know at what point we're at when
	// fetching the PSP accounts.
	// This struct will be stored as a raw json, you're free to put whatever
	// you want.
	// Example:
	// LastPage int `json:"lastPage"`
	// LastIDCreated int64 `json:"lastIDCreated"`
}

// fetchNextAccounts retrieves accounts from the Bitstamp connector configuration.
// Unlike typical PSPs, Bitstamp accounts are configured at setup time with their API credentials.
// This method returns all configured accounts in a single response (no pagination needed).
func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	newState := accountsState{}

	// Get all accounts from config (no pagination from Bitstamp API)
	pagedAccounts, err := p.client.GetAccounts(ctx, 0, req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	// Convert to PSPAccount format
	accounts := make([]models.PSPAccount, 0, len(pagedAccounts))
	for _, acc := range pagedAccounts {
		raw, err := json.Marshal(acc)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		var name *string
		if acc.Name != "" {
			name = &acc.Name
		}

		accounts = append(accounts, models.PSPAccount{
			Reference: acc.ID,
			CreatedAt: time.Now(),
			Name:      name,
			Raw:       raw,
		})
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  false,
	}, nil
}
