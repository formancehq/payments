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
	}

	accounts := make([]models.PSPAccount, 0, req.PageSize)
	needMore := false
	hasMore := false
	for page := 0; ; page++ {
		pagedBeneficiaries, err := p.client.GetBeneficiaries(ctx, page, req.PageSize, oldState.LastModifiedSince)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		accounts, err = fillBeneficiaries(pagedBeneficiaries, accounts, oldState, req.PageSize)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(accounts, pagedBeneficiaries, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	if !needMore {
		accounts = accounts[:req.PageSize]
	}

	if len(accounts) > 0 {
		newState.LastModifiedSince = accounts[len(accounts)-1].CreatedAt
		newState.LastProcessedID = accounts[len(accounts)-1].Reference
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
	pageSize int,
) ([]models.PSPAccount, error) {
	for _, beneficiary := range pagedBeneficiaries {
		if len(accounts) >= pageSize {
			break
		}

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
