package modulr

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/modulr/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type externalAccountsState struct {
	LastModifiedSince time.Time `json:"lastModifiedSince"`
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
	}

	accounts := make([]models.PSPAccount, 0, req.PageSize)
	needMore := false
	hasMore := false
	for page := 0; ; page++ {
		pageSize := req.PageSize - len(accounts)

		pagedBeneficiaries, err := p.client.GetBeneficiaries(ctx, page, pageSize, oldState.LastModifiedSince)
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

	if len(accounts) > 0 {
		newState.LastModifiedSince = accounts[len(accounts)-1].CreatedAt
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

		switch createdTime.Compare(oldState.LastModifiedSince) {
		case -1, 0:
			// Account already ingested, skip
			continue
		default:
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
