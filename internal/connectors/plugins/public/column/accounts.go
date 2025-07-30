package column

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	LastIDCreated string `json:"lastIDCreated"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	newState := accountsState{
		LastIDCreated: oldState.LastIDCreated,
	}

	accounts := make([]models.PSPAccount, 0, req.PageSize)
	hasMore := false
	pagedAccounts, hasMore, err := p.client.GetAccounts(ctx, oldState.LastIDCreated, req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accounts, err = p.fillAccounts(pagedAccounts, accounts, req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	if len(accounts) > 0 {
		newState.LastIDCreated = accounts[len(accounts)-1].Reference
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func (p *Plugin) fillAccounts(
	pagedAccounts []*client.Account,
	accounts []models.PSPAccount,
	pageSize int,
) ([]models.PSPAccount, error) {
	for _, account := range pagedAccounts {
		if len(accounts) > pageSize {
			break
		}

		createdTime, err := time.Parse(time.RFC3339, account.CreatedAt)
		if err != nil {
			return nil, err
		}

		raw, err := json.Marshal(account)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, models.PSPAccount{
			Reference:    account.ID,
			CreatedAt:    createdTime,
			Name:         &account.Description,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, account.CurrencyCode)),
			Raw:          raw,
			Metadata: map[string]string{
				client.ColumnTypeMetadataKey:                      account.Type,
				client.ColumnBicMetadataKey:                       account.Bic,
				client.ColumnDefaultAccountNumberIDMetadataKey:    account.DefaultAccountNumberID,
				client.ColumnDefaultAccountNumberMetadataKey:      account.DefaultAccountNumber,
				client.ColumnIsOverdraftableMetadataKey:           strconv.FormatBool(account.IsOverdraftable),
				client.ColumnOverdraftReserveAccountIDMetadataKey: account.OverdraftReserveAccountID,
				client.ColumnRoutingNumberMetadataKey:             account.RoutingNumber,
				client.ColumnOwnersMetadataKey:                    strings.Join(account.Owners, ","),
			},
		})
	}
	return accounts, nil
}
