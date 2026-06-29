package modulr

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/modulr/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
)

type accountsState struct {
	LastCreatedAt time.Time `json:"lastCreatedAt"`
	// LastProcessedID is the reference of the last account emitted at exactly
	// LastCreatedAt, so the inclusive (>=) watermark filter excludes only that
	// already-processed row while keeping distinct same-timestamp accounts.
	LastProcessedID string `json:"lastProcessedID"`
	// Page is the next page to fetch within the current LastCreatedAt second
	// (0-indexed). It advances while the watermark second is unchanged (a
	// same-second group larger than one page) and resets to 0 once the watermark
	// moves to a newer second, so a same-second group spanning pages is walked
	// without re-scanning from page 0 each cycle (which a single LastProcessedID
	// cannot do).
	Page int `json:"page"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	newState := accountsState{
		LastCreatedAt:   oldState.LastCreatedAt,
		LastProcessedID: oldState.LastProcessedID,
		Page:            oldState.Page,
	}

	var accounts []models.PSPAccount
	needMore := false
	hasMore := false
	// Resume at the persisted page and walk forward; the page cursor below
	// records how far we got. We consume each page fully (no PageSize cap or
	// trim) so resuming at the next page cannot skip rows.
	page := oldState.Page
	for {
		pagedAccounts, err := p.client.GetAccounts(ctx, page, req.PageSize, oldState.LastCreatedAt)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		accounts, err = fillAccounts(pagedAccounts, accounts, oldState)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(accounts, pagedAccounts, req.PageSize)
		if !needMore || !hasMore {
			break
		}
		page++
	}

	if len(accounts) > 0 {
		newState.LastCreatedAt = accounts[len(accounts)-1].CreatedAt
		newState.LastProcessedID = accounts[len(accounts)-1].Reference
		// Advance past the consumed pages only while there is definitely a full
		// next page (hasMore). If the same-second group drained on a short final
		// page, keep the cursor there — a newer row appended to that second's
		// >= watermark query lands on this very page, so advancing past it would
		// strand it forever. When the watermark moved to a newer second, re-anchor
		// at page 0.
		if newState.LastCreatedAt.Equal(oldState.LastCreatedAt) {
			if hasMore {
				newState.Page = page + 1
			} else {
				newState.Page = page
			}
		} else {
			newState.Page = 0
		}
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

func fillAccounts(
	pagedAccounts []client.Account,
	accounts []models.PSPAccount,
	oldState accountsState,
) ([]models.PSPAccount, error) {
	for _, account := range pagedAccounts {
		createdTime, err := time.Parse("2006-01-02T15:04:05.999-0700", account.CreatedDate)
		if err != nil {
			return nil, err
		}

		// Inclusive watermark: skip accounts strictly before it, and the single
		// already-processed account at exactly the watermark. Distinct accounts
		// sharing that timestamp are kept (M-CON2).
		cmp := createdTime.Compare(oldState.LastCreatedAt)
		if cmp < 0 || (cmp == 0 && account.ID == oldState.LastProcessedID) {
			continue
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
		})
	}

	return accounts, nil
}
