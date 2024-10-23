package wise

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/wise/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type externalAccountsState struct {
	LastSeekPosition uint64 `json:"lastSeekPosition"`
}

func (p *Plugin) fetchExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
	}

	var from client.Profile
	if req.FromPayload == nil {
		return models.FetchNextExternalAccountsResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	newState := externalAccountsState{
		LastSeekPosition: oldState.LastSeekPosition,
	}

	var accounts []models.PSPAccount
	needMore := false
	hasMore := false
	lastSeekPosition := oldState.LastSeekPosition
	for {
		pagedExternalAccounts, err := p.client.GetRecipientAccounts(ctx, from.ID, req.PageSize, lastSeekPosition)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		accounts, err = fillExternalAccounts(pagedExternalAccounts, accounts, oldState)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		lastSeekPosition = pagedExternalAccounts.SeekPositionForNext
		needMore, hasMore = pagination.ShouldFetchMore(accounts, pagedExternalAccounts.Content, req.PageSize)
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
		return models.FetchNextExternalAccountsResponse{}, err
	}

	return models.FetchNextExternalAccountsResponse{
		ExternalAccounts: accounts,
		NewState:         payload,
		HasMore:          hasMore,
	}, nil
}

func fillExternalAccounts(
	pagedExternalAccounts *client.RecipientAccountsResponse,
	accounts []models.PSPAccount,
	oldState externalAccountsState,
) ([]models.PSPAccount, error) {
	for _, externalAccount := range pagedExternalAccounts.Content {
		if oldState.LastSeekPosition != 0 && externalAccount.ID <= oldState.LastSeekPosition {
			continue
		}

		raw, err := json.Marshal(externalAccount)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, models.PSPAccount{
			Reference:    strconv.FormatUint(externalAccount.ID, 10),
			CreatedAt:    time.Now().UTC(),
			Name:         &externalAccount.Name.FullName,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, externalAccount.Currency)),
			Raw:          raw,
		})
	}

	return accounts, nil
}
