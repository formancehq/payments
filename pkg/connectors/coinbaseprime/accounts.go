package coinbaseprime

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connector"
)

type accountsState struct {
	Cursor string `json:"cursor"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req connector.FetchNextAccountsRequest) (connector.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}
	}

	response, err := p.client.GetWallets(ctx, oldState.Cursor, req.PageSize)
	if err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	accounts := make([]connector.PSPAccount, 0, len(response.Wallets))
	for _, wallet := range response.Wallets {
		_, ok := supportedCurrenciesWithDecimal[strings.ToUpper(wallet.Symbol)]
		if !ok {
			continue
		}

		raw, err := json.Marshal(wallet)
		if err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}

		defaultAsset := currency.FormatAsset(supportedCurrenciesWithDecimal, strings.ToUpper(wallet.Symbol))

		accounts = append(accounts, connector.PSPAccount{
			Reference:    wallet.ID,
			CreatedAt:    wallet.CreatedAt,
			Name:         &wallet.Name,
			DefaultAsset: &defaultAsset,
			Metadata: map[string]string{
				"wallet_type": wallet.Type,
			},
			Raw: raw,
		})
	}

	newState := accountsState{Cursor: response.Pagination.NextCursor}
	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	return connector.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  response.Pagination.HasNext,
	}, nil
}
