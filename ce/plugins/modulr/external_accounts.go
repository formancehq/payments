package modulr

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/ce/plugins/modulr/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
)

type externalAccountsState struct {
	LastModifiedSince time.Time `json:"lastModifiedSince"`
	// LastProcessedID is the reference of the last beneficiary emitted at exactly
	// LastModifiedSince, so the inclusive (>=) watermark filter excludes only that
	// already-processed row while keeping distinct same-timestamp beneficiaries.
	LastProcessedID string `json:"lastProcessedID"`
	// Page is the next page to fetch within the current LastModifiedSince second
	// (0-indexed). It advances while the watermark second is unchanged (a
	// same-second group larger than one page) and resets to 0 once the watermark
	// moves to a newer second, so a same-second group spanning pages is walked
	// without re-scanning from page 0 each cycle (which a single LastProcessedID
	// cannot do).
	Page int `json:"page"`
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
		LastProcessedID:   oldState.LastProcessedID,
		Page:              oldState.Page,
	}

	accounts := make([]models.PSPAccount, 0, req.PageSize)
	needMore := false
	hasMore := false
	// Resume at the persisted page and walk forward; the page cursor below
	// records how far we got. We consume each page fully (no PageSize cap or
	// trim) so resuming at the next page cannot skip rows.
	page := oldState.Page
	for {
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
		page++
	}

	if len(accounts) > 0 {
		newState.LastModifiedSince = accounts[len(accounts)-1].CreatedAt
		newState.LastProcessedID = accounts[len(accounts)-1].Reference
		// Advance past the consumed pages only while there is definitely a full
		// next page (hasMore). If the same-second group drained on a short final
		// page, keep the cursor there — a newer row appended to that second's
		// >= watermark query lands on this very page, so advancing past it would
		// strand it forever. When the watermark moved to a newer second, re-anchor
		// at page 0.
		if newState.LastModifiedSince.Equal(oldState.LastModifiedSince) {
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

		// Inclusive watermark: skip beneficiaries strictly before it, and the single
		// already-processed beneficiary at exactly the watermark. Distinct
		// beneficiaries sharing that timestamp are kept (M-CON2).
		cmp := createdTime.Compare(oldState.LastModifiedSince)
		if cmp < 0 || (cmp == 0 && beneficiary.ID == oldState.LastProcessedID) {
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
