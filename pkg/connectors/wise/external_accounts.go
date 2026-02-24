package wise

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/pkg/connectors/wise/client"
	"github.com/formancehq/payments/pkg/connector"
)

type externalAccountsState struct {
	LastSeekPosition uint64 `json:"lastSeekPosition"`
}

func (p *Plugin) fetchExternalAccounts(ctx context.Context, req connector.FetchNextExternalAccountsRequest) (connector.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}
	}

	var from client.Profile
	if req.FromPayload == nil {
		return connector.FetchNextExternalAccountsResponse{}, connector.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return connector.FetchNextExternalAccountsResponse{}, err
	}

	newState := externalAccountsState{
		LastSeekPosition: oldState.LastSeekPosition,
	}

	var accounts []connector.PSPAccount
	needMore := false
	hasMore := false
	lastSeekPosition := oldState.LastSeekPosition
	for {
		pagedExternalAccounts, err := p.client.GetRecipientAccounts(ctx, from.ID, req.PageSize, lastSeekPosition)
		if err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}

		accounts, err = fillExternalAccounts(pagedExternalAccounts, accounts, oldState)
		if err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}

		lastSeekPosition = pagedExternalAccounts.SeekPositionForNext
		needMore, hasMore = connector.ShouldFetchMore(accounts, pagedExternalAccounts.Content, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	if !needMore {
		accounts = accounts[:req.PageSize]
	}

	if len(accounts) > 0 {
		// No need to check the error, it's already checked in the fillExternalAccounts function
		id, _ := strconv.ParseUint(accounts[len(accounts)-1].Reference, 10, 64)
		newState.LastSeekPosition = id
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

func fillExternalAccounts(
	pagedExternalAccounts *client.RecipientAccountsResponse,
	accounts []connector.PSPAccount,
	oldState externalAccountsState,
) ([]connector.PSPAccount, error) {
	for _, externalAccount := range pagedExternalAccounts.Content {
		if oldState.LastSeekPosition != 0 && externalAccount.ID <= oldState.LastSeekPosition {
			continue
		}

		raw, err := json.Marshal(externalAccount)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, connector.PSPAccount{
			Reference:    strconv.FormatUint(externalAccount.ID, 10),
			CreatedAt:    time.Now().UTC(),
			Name:         &externalAccount.Name.FullName,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, externalAccount.Currency)),
			Raw:          raw,
		})
	}

	return accounts, nil
}
