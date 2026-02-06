package modulr

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/pkg/connectors/modulr/client"
	"github.com/formancehq/payments/pkg/connector"
)

type externalAccountsState struct {
	LastModifiedSince time.Time `json:"lastModifiedSince"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req connector.FetchNextExternalAccountsRequest) (connector.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}
	}

	newState := externalAccountsState{
		LastModifiedSince: oldState.LastModifiedSince,
	}

	accounts := make([]connector.PSPAccount, 0, req.PageSize)
	needMore := false
	hasMore := false
	for page := 0; ; page++ {
		pagedBeneficiaries, err := p.client.GetBeneficiaries(ctx, page, req.PageSize, oldState.LastModifiedSince)
		if err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}

		accounts, err = fillBeneficiaries(pagedBeneficiaries, accounts, oldState, req.PageSize)
		if err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}

		needMore, hasMore = connector.ShouldFetchMore(accounts, pagedBeneficiaries, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	if !needMore {
		accounts = accounts[:req.PageSize]
	}

	if len(accounts) > 0 {
		newState.LastModifiedSince = accounts[len(accounts)-1].CreatedAt
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextExternalAccountsResponse{}, err
	}

	return connector.FetchNextExternalAccountsResponse{
		ExternalAccounts: accounts,
		NewState:         payload,
		HasMore:          hasMore,
	}, nil
}

func fillBeneficiaries(
	pagedBeneficiaries []client.Beneficiary,
	accounts []connector.PSPAccount,
	oldState externalAccountsState,
	pageSize int,
) ([]connector.PSPAccount, error) {
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

		accounts = append(accounts, connector.PSPAccount{
			Reference: beneficiary.ID,
			CreatedAt: createdTime,
			Name:      &beneficiary.Name,
			Raw:       raw,
		})
	}

	return accounts, nil
}
