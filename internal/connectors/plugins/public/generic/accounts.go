package generic

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/formancehq/payments/genericclient/v3"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
)

type accountsState struct {
	LastCreatedAtFrom time.Time `json:"lastCreatedAtFrom"`
	// LastProcessedIDs holds the references of ALL accounts already emitted at
	// exactly LastCreatedAtFrom, accumulated across cycles while the watermark
	// second is unchanged and reset when it advances. The server filter is
	// inclusive (>=), so each cycle rescans from page 1 and skips this whole set:
	// a same-second group larger than PageSize is walked across cycles without a
	// drifting page cursor, and a multi-row final page cannot oscillate (every
	// already-emitted sibling is skipped, not just one).
	//
	// Migration note: the old singular lastProcessedID and page fields are
	// ignored on first decode after deploy, so the watermark second's group is
	// re-emitted once. This is idempotent via storage upsert — no recrawl.
	//
	// Precision: comparison and the ID set use the exact timestamp the API
	// returns (full precision, as in the qonto reference), not a truncated
	// second; "same-second" above is shorthand because these PSPs report
	// timestamps at second granularity, so equal values represent the same second.
	LastProcessedIDs []string `json:"lastProcessedIDs"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	newState := accountsState{
		LastCreatedAtFrom: oldState.LastCreatedAtFrom,
		LastProcessedIDs:  oldState.LastProcessedIDs,
	}

	accounts := make([]models.PSPAccount, 0)
	createdAts := make([]time.Time, 0)
	needMore := false
	hasMore := false
	// Rescan from page 1 each cycle (no page cursor): the processed-ID set skips
	// every already-emitted sibling at the watermark second, so a same-second
	// group larger than PageSize is walked across cycles and a multi-row final
	// page cannot oscillate. The server filter is inclusive (>=), so page 1
	// re-includes the watermark second.
	for page := 1; ; page++ {
		pagedAccounts, err := p.client.ListAccounts(ctx, int64(page), int64(req.PageSize), oldState.LastCreatedAtFrom)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		accounts, createdAts, err = fillAccounts(pagedAccounts, accounts, createdAts, oldState)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(accounts, pagedAccounts, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	if len(createdAts) > 0 {
		lastCreatedAt := createdAts[len(createdAts)-1]

		// Collect the references emitted at exactly the new watermark second.
		idsAtWatermark := make([]string, 0)
		for i := range accounts {
			if createdAts[i].Equal(lastCreatedAt) {
				idsAtWatermark = append(idsAtWatermark, accounts[i].Reference)
			}
		}

		// Accumulate the processed-ID set while still inside the same watermark
		// second; reset it when the watermark advances to a newer second.
		if lastCreatedAt.Equal(oldState.LastCreatedAtFrom) {
			newState.LastProcessedIDs = append(oldState.LastProcessedIDs, idsAtWatermark...)
		} else {
			newState.LastProcessedIDs = idsAtWatermark
		}
		newState.LastCreatedAtFrom = lastCreatedAt
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
	pagedAccounts []genericclient.Account,
	accounts []models.PSPAccount,
	createdAts []time.Time,
	oldState accountsState,
) ([]models.PSPAccount, []time.Time, error) {
	for _, account := range pagedAccounts {
		// Inclusive watermark: skip accounts strictly before it, and any already-
		// emitted account at exactly the watermark second. Distinct accounts
		// sharing that timestamp are kept (M-CON2).
		cmp := account.CreatedAt.Compare(oldState.LastCreatedAtFrom)
		if cmp < 0 || (cmp == 0 && slices.Contains(oldState.LastProcessedIDs, account.Id)) {
			continue
		}

		raw, err := json.Marshal(account)
		if err != nil {
			return nil, nil, err
		}

		accounts = append(accounts, models.PSPAccount{
			Reference: account.Id,
			CreatedAt: account.CreatedAt,
			Name:      &account.AccountName,
			Metadata:  account.Metadata,
			Raw:       raw,
		})
		createdAts = append(createdAts, account.CreatedAt)
	}

	return accounts, createdAts, nil
}
