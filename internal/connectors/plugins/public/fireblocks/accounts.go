package fireblocks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/fireblocks/client"
	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	After string `json:"after,omitempty"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}

	params := client.GetVaultAccountsParams{
		Limit: PAGE_SIZE,
		After: oldState.After,
	}

	resp, err := p.client.GetVaultAccounts(ctx, params)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to get vault accounts: %w", err)
	}

	accounts := make([]models.PSPAccount, 0, len(resp.Accounts))
	for _, vaultAccount := range resp.Accounts {
		raw, err := json.Marshal(vaultAccount)
		if err != nil {
			return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to marshal vault account: %w", err)
		}

		metadata := map[string]string{
			"vault_id":   vaultAccount.ID,
			"vault_name": vaultAccount.Name,
		}
		if vaultAccount.CustomerRefID != "" {
			metadata["customer_ref_id"] = vaultAccount.CustomerRefID
		}
		if vaultAccount.AutoFuel {
			metadata["auto_fuel"] = "true"
		}
		if vaultAccount.HiddenOnUI {
			metadata["hidden_on_ui"] = "true"
		}

		account := models.PSPAccount{
			Reference: vaultAccount.ID,
			Name:      &vaultAccount.Name,
			CreatedAt: time.Now().UTC(),
			Metadata:  metadata,
			Raw:       raw,
		}

		accounts = append(accounts, account)

		// Also create sub-accounts for each asset in the vault
		for _, asset := range vaultAccount.Assets {
			assetRaw, _ := json.Marshal(asset)
			assetRef := fmt.Sprintf("%s-%s", vaultAccount.ID, asset.ID)
			assetName := fmt.Sprintf("%s - %s", vaultAccount.Name, asset.ID)

			assetMetadata := map[string]string{
				"vault_id":   vaultAccount.ID,
				"vault_name": vaultAccount.Name,
				"asset_id":   asset.ID,
			}

			assetAccount := models.PSPAccount{
				Reference: assetRef,
				Name:      &assetName,
				CreatedAt: time.Now().UTC(),
				Metadata:  assetMetadata,
				Raw:       assetRaw,
			}

			accounts = append(accounts, assetAccount)
		}
	}

	// Determine if there are more results
	hasMore := resp.Paging.After != ""

	newState := accountsState{
		After: resp.Paging.After,
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
