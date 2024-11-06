package atlar

import (
	"context"
	"encoding/json"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/models"
	"github.com/get-momo/atlar-v1-go-client/client/accounts"
)

type accountsState struct {
	NextToken string `json:"nextToken"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	var accounts []models.PSPAccount
	nextToken := oldState.NextToken
	for {
		resp, err := p.client.GetV1Accounts(ctx, nextToken, int64(req.PageSize))
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		accounts, err = p.fillAccounts(ctx, resp, accounts)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		nextToken = resp.Payload.NextToken
		if resp.Payload.NextToken == "" || len(accounts) >= req.PageSize {
			break
		}
	}

	// If token is empty, this is perfect as the next polling task will refetch
	// everything ! And that's what we want since Atlar doesn't provide any
	// filters or sorting options.
	newState := accountsState{
		NextToken: nextToken,
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  nextToken != "",
	}, nil
}

func (p *Plugin) fillAccounts(
	ctx context.Context,
	pagedAccounts *accounts.GetV1AccountsOK,
	accounts []models.PSPAccount,
) ([]models.PSPAccount, error) {
	for _, account := range pagedAccounts.Payload.Items {
		raw, err := json.Marshal(account)
		if err != nil {
			return nil, err
		}

		createdAt, err := ParseAtlarTimestamp(account.Created)
		if err != nil {
			return nil, err
		}

		thirdPartyResponse, err := p.client.GetV1BetaThirdPartiesID(ctx, account.ThirdPartyID)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, models.PSPAccount{
			Reference:    *account.ID,
			CreatedAt:    createdAt,
			Name:         &account.Name,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, account.Currency)),
			Metadata:     ExtractAccountMetadata(account, thirdPartyResponse.Payload),
			Raw:          raw,
		})
	}

	return accounts, nil
}
