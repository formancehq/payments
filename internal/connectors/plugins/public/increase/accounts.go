package increase

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	LastID   string          `json:"last_id"`
	Timeline json.RawMessage `json:"timeline"`
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var state accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}

	accounts, nextCursor, hasMore, err := p.client.GetAccounts(ctx, state.LastID, int64(req.PageSize))
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to get accounts: %w", err)
	}

	pspAccounts := make([]models.PSPAccount, len(accounts))
	for i, account := range accounts {
		raw, err := json.Marshal(account)
		if err != nil {
			return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to marshal account: %w", err)
		}

		pspAccounts[i] = models.PSPAccount{
			ID:        account.ID,
			CreatedAt: account.CreatedAt,
			Reference: account.ID,
			Type:      models.AccountType(account.Type),
			Status:    models.AccountStatus(account.Status),
			Raw:       raw,
		}
	}

	newState := accountsState{
		LastID: nextCursor,
	}
	newStateBytes, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to marshal new state: %w", err)
	}

	return models.FetchNextAccountsResponse{
		Accounts: pspAccounts,
		NewState: newStateBytes,
		HasMore:  hasMore,
	}, nil
}
