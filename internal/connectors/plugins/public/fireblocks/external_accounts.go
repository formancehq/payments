package fireblocks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	// Fireblocks doesn't support pagination for external wallets, so we fetch all at once
	externalWallets, err := p.client.GetExternalWallets(ctx)
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, fmt.Errorf("failed to get external wallets: %w", err)
	}

	accounts := make([]models.PSPAccount, 0, len(externalWallets))
	now := time.Now().UTC()

	for _, wallet := range externalWallets {
		raw, err := json.Marshal(wallet)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, fmt.Errorf("failed to marshal external wallet: %w", err)
		}

		metadata := map[string]string{
			"wallet_id":   wallet.ID,
			"wallet_name": wallet.Name,
			"wallet_type": "external",
		}
		if wallet.CustomerRefID != "" {
			metadata["customer_ref_id"] = wallet.CustomerRefID
		}

		account := models.PSPAccount{
			Reference: fmt.Sprintf("external-%s", wallet.ID),
			Name:      &wallet.Name,
			CreatedAt: now,
			Metadata:  metadata,
			Raw:       raw,
		}

		accounts = append(accounts, account)

		// Also add sub-accounts for each asset in the wallet
		for _, asset := range wallet.Assets {
			assetRaw, _ := json.Marshal(asset)
			assetRef := fmt.Sprintf("external-%s-%s", wallet.ID, asset.ID)
			assetName := fmt.Sprintf("%s - %s", wallet.Name, asset.ID)

			assetMetadata := map[string]string{
				"wallet_id":   wallet.ID,
				"wallet_name": wallet.Name,
				"wallet_type": "external",
				"asset_id":    asset.ID,
				"address":     asset.Address,
				"status":      asset.Status,
			}
			if asset.Tag != "" {
				assetMetadata["tag"] = asset.Tag
			}

			assetAccount := models.PSPAccount{
				Reference: assetRef,
				Name:      &assetName,
				CreatedAt: now,
				Metadata:  assetMetadata,
				Raw:       assetRaw,
			}

			accounts = append(accounts, assetAccount)
		}
	}

	// Also fetch internal wallets
	internalWallets, err := p.client.GetInternalWallets(ctx)
	if err == nil {
		for _, wallet := range internalWallets {
			raw, _ := json.Marshal(wallet)

			metadata := map[string]string{
				"wallet_id":   wallet.ID,
				"wallet_name": wallet.Name,
				"wallet_type": "internal",
			}
			if wallet.CustomerRefID != "" {
				metadata["customer_ref_id"] = wallet.CustomerRefID
			}

			account := models.PSPAccount{
				Reference: fmt.Sprintf("internal-%s", wallet.ID),
				Name:      &wallet.Name,
				CreatedAt: now,
				Metadata:  metadata,
				Raw:       raw,
			}

			accounts = append(accounts, account)
		}
	}

	return models.FetchNextExternalAccountsResponse{
		ExternalAccounts: accounts,
		HasMore:          false,
	}, nil
}
