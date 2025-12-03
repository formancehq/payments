package bitstamp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
)

// accountsState tracks pagination state for fetching accounts.
type accountsState struct {
	Page int `json:"page"`
}

// fetchNextAccounts retrieves accounts from the Bitstamp connector configuration.
// Unlike typical PSPs, Bitstamp accounts are configured at setup time with their API credentials.
// This method returns configured accounts with pagination support.
func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var state accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	// Get paged accounts from config
	pagedAccounts, err := p.client.GetAccounts(ctx, state.Page, req.PageSize)
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

		pspAccount := models.PSPAccount{
			Reference: acc.ID,
			CreatedAt: time.Now(),
			Name:      name,
			Raw:       raw,
		}
		p.logger.Infof("Creating PSPAccount with reference: %s (name: %s)", pspAccount.Reference, acc.Name)
		accounts = append(accounts, pspAccount)
	}

	p.logger.Infof("Returning %d accounts for storage", len(accounts))

	hasMore := len(pagedAccounts) == req.PageSize
	if hasMore {
		state.Page++
	}

	payload, err := json.Marshal(state)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}
