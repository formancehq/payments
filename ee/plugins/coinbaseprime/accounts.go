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

	// Merge the returned wallets into p.wallets as a side effect of pagination.
	// Orders rely on p.wallets[symbol] to resolve SourceAccountReference /
	// DestinationAccountReference. We update even when the currency is
	// unsupported so the wallet ID is available if support is added later via
	// a subsequent ensureAssetsFresh refresh. Never clear — deleted wallets
	// remain mapped until the plugin instance is recreated.
	p.assetsMu.Lock()
	if p.wallets == nil {
		p.wallets = make(map[string]string, len(response.Wallets))
	}
	for _, wallet := range response.Wallets {
		sym := strings.ToUpper(strings.TrimSpace(wallet.Symbol))
		if sym != "" {
			p.wallets[sym] = wallet.ID
		}
	}
	// Snapshot currencies under the same lock to keep the filter below
	// consistent with the side-effect update above.
	currencies := p.currencies
	p.assetsMu.Unlock()

	accounts := make([]models.PSPAccount, 0, len(response.Wallets))
	for _, wallet := range response.Wallets {
		symbol := strings.ToUpper(strings.TrimSpace(wallet.Symbol))
		_, ok := currencies[symbol]
		if !ok {
			p.logger.Infof("skipping wallet %s: unsupported currency %q", wallet.ID, wallet.Symbol)
			continue
		}

		raw, err := json.Marshal(wallet)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		defaultAsset := currency.FormatAsset(currencies, symbol)

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
