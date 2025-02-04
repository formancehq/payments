package increase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/models"
	"github.com/Increase/increase-go"
)

type pollingState struct {
	NextCursor string    `json:"next_cursor"`
	LastFetch  time.Time `json:"last_fetch"`
}

func (p *Plugin) getPollingState(state json.RawMessage) (*pollingState, error) {
	if len(state) == 0 {
		return &pollingState{
			LastFetch: time.Now().UTC(),
		}, nil
	}

	var s pollingState
	if err := json.Unmarshal(state, &s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}
	return &s, nil
}

func (p *Plugin) mapAccount(a *increase.Account) (*models.PSPAccount, error) {
	raw, err := json.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal account: %w", err)
	}

	return &models.PSPAccount{
		Reference:    a.ID,
		CreatedAt:    a.CreatedAt,
		Name:         &a.Name,
		DefaultAsset: pointer.String("USD"),
		Metadata: map[string]string{
			"status":   string(a.Status),
			"type":     string(a.Type),
			"bank":     string(a.Bank),
			"currency": string(a.Currency),
		},
		Raw: raw,
	}, nil
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	state, err := p.getPollingState(req.State)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to get polling state: %w", err)
	}

	if state.LastFetch.Add(p.config.PollingPeriod).After(time.Now().UTC()) {
		return models.FetchNextAccountsResponse{}, nil
	}

	accounts, nextCursor, hasMore, err := p.client.GetAccounts(ctx, state.NextCursor, int64(req.PageSize))
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to get accounts: %w", err)
	}

	pspAccounts := make([]models.PSPAccount, len(accounts))
	for i, account := range accounts {
		pspAccount, err := p.mapAccount(account)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
		pspAccounts[i] = *pspAccount
	}

	newState := pollingState{
		NextCursor: nextCursor,
		LastFetch:  time.Now().UTC(),
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
