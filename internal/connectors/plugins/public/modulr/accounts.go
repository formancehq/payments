package modulr

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/modulr/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
)

type accountsState struct {
	LastCreatedAt time.Time `json:"lastCreatedAt"`
	// LastProcessedIDs holds the references of ALL accounts already emitted at
	// exactly LastCreatedAt, accumulated across cycles while the watermark second
	// is unchanged and reset when it advances. The server filter is inclusive
	// (>=), so each cycle rescans from page 0 and skips this whole set: a
	// same-second group larger than PageSize is walked across cycles without a
	// drifting page cursor, and a multi-row final page cannot oscillate (every
	// already-emitted sibling is skipped, not just one).
	//
	// Migration note: the previous schema used a singular LastProcessedID plus a
	// Page cursor; both are ignored on the first decode after deploy (the old
	// JSON keys simply don't bind). The watermark second is re-emitted once after
	// deploy, which is idempotent (storage upserts dedup it) and triggers no
	// recrawl.
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
		LastCreatedAt:    oldState.LastCreatedAt,
		LastProcessedIDs: oldState.LastProcessedIDs,
	}

	var accounts []models.PSPAccount
	needMore := false
	hasMore := false
	// Rescan from page 0 each cycle (no page cursor): the processed-ID set skips
	// every already-emitted sibling at the watermark second, so a same-second
	// group larger than PageSize is walked across cycles and a multi-row final
	// page cannot oscillate. The server filter is inclusive (>=), so page 0
	// re-includes the watermark second.
	for page := 0; ; page++ {
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
	}

	if len(accounts) > 0 {
		last := accounts[len(accounts)-1].CreatedAt

		// Collect the references emitted at exactly the new watermark second.
		idsAtWatermark := make([]string, 0)
		for i := range accounts {
			if accounts[i].CreatedAt.Equal(last) {
				idsAtWatermark = append(idsAtWatermark, accounts[i].Reference)
			}
		}

		// Accumulate the processed-ID set while still inside the same watermark
		// second; reset it when the watermark advances to a newer second.
		if last.Equal(oldState.LastCreatedAt) {
			newState.LastProcessedIDs = append(oldState.LastProcessedIDs, idsAtWatermark...)
		} else {
			newState.LastProcessedIDs = idsAtWatermark
		}
		newState.LastCreatedAt = last
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

		// Inclusive watermark: skip accounts strictly before it, and any already-
		// emitted account at exactly the watermark second. Distinct accounts
		// sharing that timestamp are kept (M-CON2).
		cmp := createdTime.Compare(oldState.LastCreatedAt)
		if cmp < 0 || (cmp == 0 && slices.Contains(oldState.LastProcessedIDs, account.ID)) {
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
