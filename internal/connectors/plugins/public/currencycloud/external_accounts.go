package currencycloud

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/currencycloud/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

type externalAccountsState struct {
	// LastPage is a monotonic forward cursor. This endpoint has NO server-side
	// time filter, so resuming the scan here (re-reading only the last page, not
	// the whole history each poll) avoids a full historical rescan on every sync.
	LastPage      int       `json:"lastPage"`
	LastCreatedAt time.Time `json:"lastCreatedAt"`
	// LastProcessedIDs holds the IDs of ALL beneficiaries already emitted at
	// exactly LastCreatedAt, accumulated while the watermark second is unchanged
	// and reset when it advances. It dedups same-second rows on the re-read page,
	// so a multi-row boundary settles to empty instead of oscillating (a single
	// LastProcessedID could exclude only one of several tied rows).
	//
	// Migration: the old singular lastProcessedID is ignored; the watermark second
	// is re-emitted once after deploy (idempotent via storage upserts), no recrawl.
	//
	// Precision: comparison and the ID set use the exact timestamp the API
	// returns (full precision, as in the qonto reference), not a truncated
	// second; "same-second" above is shorthand because these PSPs report
	// timestamps at second granularity, so equal values represent the same second.
	LastProcessedIDs []string `json:"lastProcessedIDs"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
	}

	if oldState.LastPage == 0 {
		oldState.LastPage = 1
	}

	newState := externalAccountsState{
		LastPage:         oldState.LastPage,
		LastCreatedAt:    oldState.LastCreatedAt,
		LastProcessedIDs: oldState.LastProcessedIDs,
	}

	var accounts []models.PSPAccount
	hasMore := false
	// No server-side time filter: resume the monotonic forward scan at the
	// persisted page (re-reading only the last page, not the whole history) and
	// skip already-emitted same-second rows via the ID set, so a multi-row final
	// page cannot oscillate. We do NOT trim back to PageSize.
	page := oldState.LastPage
	for {
		pagedBeneficiaries, nextPage, err := p.client.GetBeneficiaries(ctx, page, req.PageSize)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		if len(pagedBeneficiaries) == 0 {
			break
		}

		accounts, err = fillBeneficiaries(accounts, pagedBeneficiaries, oldState)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		var needMore bool
		// Ignore shouldFetchMore's trimmed slice; keep the full over-fetch.
		needMore, hasMore, _ = shouldFetchMore(accounts, nextPage, req.PageSize)
		if !needMore {
			break
		}
		page = nextPage
	}
	newState.LastPage = page

	if len(accounts) > 0 {
		last := accounts[len(accounts)-1].CreatedAt

		// Collect the IDs emitted at exactly the new watermark second.
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
		return models.FetchNextExternalAccountsResponse{}, err
	}

	return models.FetchNextExternalAccountsResponse{
		ExternalAccounts: accounts,
		NewState:         payload,
		HasMore:          hasMore,
	}, nil
}

func fillBeneficiaries(
	accounts []models.PSPAccount,
	pagedBeneficiaries []*client.Beneficiary,
	oldState externalAccountsState,
) ([]models.PSPAccount, error) {
	for _, beneficiary := range pagedBeneficiaries {
		// Inclusive watermark: skip beneficiaries strictly before it, and any
		// already-emitted beneficiary at exactly the watermark second. Distinct
		// beneficiaries sharing that timestamp are kept (M-CON2).
		cmp := beneficiary.CreatedAt.Compare(oldState.LastCreatedAt)
		if cmp < 0 || (cmp == 0 && slices.Contains(oldState.LastProcessedIDs, beneficiary.ID)) {
			continue
		}

		raw, err := json.Marshal(beneficiary)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, models.PSPAccount{
			Reference:    beneficiary.ID,
			CreatedAt:    beneficiary.CreatedAt,
			Name:         &beneficiary.Name,
			DefaultAsset: &beneficiary.Currency,
			Raw:          raw,
		})
	}

	return accounts, nil
}
