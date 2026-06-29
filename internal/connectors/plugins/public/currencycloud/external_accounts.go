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
	LastCreatedAt time.Time `json:"lastCreatedAt"`
	// LastProcessedIDs holds the IDs of ALL beneficiaries already emitted at
	// exactly LastCreatedAt, accumulated across cycles while the watermark second
	// is unchanged and reset when it advances. Each cycle rescans from page 1 and
	// skips this whole set: a same-second group larger than PageSize is walked
	// across cycles without a drifting page cursor, and a multi-row final page
	// cannot oscillate (every already-emitted sibling is skipped, not just one).
	//
	// Migration: the old singular lastProcessedID and lastPage fields are ignored.
	// After deploy the watermark second is re-emitted once (idempotent via storage
	// upserts), with no recrawl.
	LastProcessedIDs []string `json:"lastProcessedIDs"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
	}

	newState := externalAccountsState{
		LastCreatedAt:    oldState.LastCreatedAt,
		LastProcessedIDs: oldState.LastProcessedIDs,
	}

	var accounts []models.PSPAccount
	hasMore := false
	// Rescan from page 1 each cycle (no page cursor): the processed-ID set skips
	// every already-emitted sibling at the watermark second, so a same-second
	// group larger than PageSize is walked across cycles and a multi-row final
	// page cannot oscillate. We do NOT trim back to PageSize.
	for page := 1; ; page++ {
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
	}

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
