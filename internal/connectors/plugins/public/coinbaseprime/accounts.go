package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coinbase-samples/prime-sdk-go/model"
	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	Cursor string `json:"cursor"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var state accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}

	walletsResp, err := p.client.GetWallets(ctx, state.Cursor, req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to get wallets: %w", err)
	}

	accounts := make([]models.PSPAccount, 0, len(walletsResp.Wallets))
	for _, wallet := range walletsResp.Wallets {
		account := walletToAccount(wallet)
		accounts = append(accounts, account)
	}

	var newCursor string
	hasMore := false
	if walletsResp.Pagination != nil {
		newCursor = walletsResp.Pagination.NextCursor
		hasMore = walletsResp.Pagination.HasNext
	}

	newState := accountsState{
		Cursor: newCursor,
	}

	stateBytes, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to marshal state: %w", err)
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: stateBytes,
		HasMore:  hasMore,
	}, nil
}

func walletToAccount(wallet *model.Wallet) models.PSPAccount {
	raw, _ := json.Marshal(wallet)

	metadata := map[string]string{
		"wallet_type": wallet.Type,
		"symbol":      wallet.Symbol,
		"visibility":  string(wallet.Visibility),
	}

	if wallet.Network != nil {
		metadata["network_id"] = wallet.Network.Id
		metadata["network_type"] = wallet.Network.Type
	}

	if wallet.Address != "" {
		metadata["address"] = wallet.Address
	}

	createdAt := wallet.Created
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	return models.PSPAccount{
		Reference: wallet.Id,
		Name:      &wallet.Name,
		CreatedAt: createdAt,
		Metadata:  metadata,
		Raw:       raw,
	}
}
