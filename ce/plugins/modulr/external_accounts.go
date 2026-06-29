package modulr

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/formancehq/payments/ce/plugins/modulr/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
)

type externalAccountsState struct {
	LastModifiedSince time.Time `json:"lastModifiedSince"`
	// LastProcessedIDs holds the references of ALL beneficiaries already emitted
	// at exactly LastModifiedSince, accumulated across cycles while the watermark
	// second is unchanged and reset when it advances. The server filter is
	// inclusive (>=), so each cycle rescans from page 0 and skips this whole set:
	// a same-second group larger than PageSize is walked across cycles without a
	// drifting page cursor, and a multi-row final page cannot oscillate (every
	// already-emitted sibling is skipped, not just one).
	//
	// Migration note: the previous schema used a singular LastProcessedID plus a
	// Page cursor; both are ignored on the first decode after deploy (the old
	// JSON keys simply don't bind). The watermark second is re-emitted once after
	// deploy, which is idempotent (storage upserts dedup it) and triggers no
	// recrawl.
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
		LastModifiedSince: oldState.LastModifiedSince,
		LastProcessedIDs:  oldState.LastProcessedIDs,
	}

	accounts := make([]models.PSPAccount, 0, req.PageSize)
	needMore := false
	hasMore := false
	// Rescan from page 0 each cycle (no page cursor): the processed-ID set skips
	// every already-emitted sibling at the watermark second, so a same-second
	// group larger than PageSize is walked across cycles and a multi-row final
	// page cannot oscillate. The server filter is inclusive (>=), so page 0
	// re-includes the watermark second.
	for page := 0; ; page++ {
		pagedBeneficiaries, err := p.client.GetBeneficiaries(ctx, page, req.PageSize, oldState.LastModifiedSince)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		accounts, err = fillBeneficiaries(pagedBeneficiaries, accounts, oldState)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(accounts, pagedBeneficiaries, req.PageSize)
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
		if last.Equal(oldState.LastModifiedSince) {
			newState.LastProcessedIDs = append(oldState.LastProcessedIDs, idsAtWatermark...)
		} else {
			newState.LastProcessedIDs = idsAtWatermark
		}
		newState.LastModifiedSince = last
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
	pagedBeneficiaries []client.Beneficiary,
	accounts []models.PSPAccount,
	oldState externalAccountsState,
) ([]models.PSPAccount, error) {
	for _, beneficiary := range pagedBeneficiaries {
		createdTime, err := time.Parse("2006-01-02T15:04:05.999-0700", beneficiary.Created)
		if err != nil {
			return nil, err
		}

		// Inclusive watermark: skip beneficiaries strictly before it, and any
		// already-emitted beneficiary at exactly the watermark second. Distinct
		// beneficiaries sharing that timestamp are kept (M-CON2).
		cmp := createdTime.Compare(oldState.LastModifiedSince)
		if cmp < 0 || (cmp == 0 && slices.Contains(oldState.LastProcessedIDs, beneficiary.ID)) {
			continue
		}

		raw, err := json.Marshal(beneficiary)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, models.PSPAccount{
			Reference: beneficiary.ID,
			CreatedAt: createdTime,
			Name:      &beneficiary.Name,
			Raw:       raw,
		})
	}

	return accounts, nil
}
