package coinbaseprime

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	Cursor string `json:"cursor"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	response, err := p.client.GetWallets(ctx, oldState.Cursor, req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accounts := make([]models.PSPAccount, 0, len(response.Wallets))
	for _, wallet := range response.Wallets {
		symbol := strings.ToUpper(strings.TrimSpace(wallet.Symbol))
		_, ok := p.currencies[symbol]
		if !ok {
			p.logger.Infof("skipping wallet %s: unsupported currency %q", wallet.ID, wallet.Symbol)
			continue
		}

		raw, err := json.Marshal(wallet)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		defaultAsset := currency.FormatAsset(p.currencies, symbol)

		accounts = append(accounts, models.PSPAccount{
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
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  response.Pagination.HasNext,
	}, nil
}
