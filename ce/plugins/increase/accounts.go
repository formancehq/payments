package increase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/payments/ce/plugins/increase/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

type accountsState struct {
	NextCursor    string    `json:"next_cursor"`
	LastCreatedAt time.Time `json:"last_created_at"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	accounts := make([]models.PSPAccount, 0, req.PageSize)
	hasMore := false
	pagedAccounts, nextCursor, err := p.client.GetAccounts(ctx, req.PageSize, oldState.NextCursor, oldState.LastCreatedAt)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accounts, err = p.fillAccounts(pagedAccounts, accounts, req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	hasMore = nextCursor != ""

	newState := accountsState{
		NextCursor:    nextCursor,
		LastCreatedAt: oldState.LastCreatedAt,
	}

	// Increase returns accounts newest-first (reverse chronological), so the
	// greatest created_at is the first element of the page. Advance the
	// watermark monotonically off that max and never let a later (older) cursor
	// page drag it back, so a fresh incremental poll queries created_at.after
	// the newest account seen rather than the oldest.
	if len(accounts) > 0 && accounts[0].CreatedAt.After(newState.LastCreatedAt) {
		newState.LastCreatedAt = accounts[0].CreatedAt
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func (p *Plugin) fillAccounts(
	pagedAccounts []*client.Account,
	accounts []models.PSPAccount,
	pageSize int,
) ([]models.PSPAccount, error) {
	for _, account := range pagedAccounts {
		if len(accounts) >= pageSize {
			break
		}

		createdTime, err := time.Parse(time.RFC3339, account.CreatedAt)
		if err != nil {
			return nil, err
		}

		raw, err := json.Marshal(account)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, models.PSPAccount{
			Reference:    account.ID,
			CreatedAt:    createdTime,
			Name:         &account.Name,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, account.Currency)),
			Raw:          raw,
			Metadata: map[string]string{
				client.IncreaseTypeMetadataKey:   account.Type,
				client.IncreaseBankMetadataKey:   account.Bank,
				client.IncreaseStatusMetadataKey: account.Status,
			},
		})
	}

	return accounts, nil
}
