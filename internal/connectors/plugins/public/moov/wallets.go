package moov

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
	"github.com/moovfinancial/moov-go/pkg/moov"
)

type walletsState struct {
	AccountIndex int      `json:"account_index"`
	AccountID    string   `json:"account_id"`
	Skip         int      `json:"skip"`
	AccountIDs   []string `json:"account_ids"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState walletsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	// If we don't have account IDs yet, we need to get them from the request
	if len(oldState.AccountIDs) == 0 && req.FromPayload != nil {
		var from models.PSPOther
		if err := json.Unmarshal(req.FromPayload, &from); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		var account moov.Account
		if err := json.Unmarshal(from.Other, &account); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		oldState.AccountIDs = []string{account.ID}
	}

	// If we still don't have account IDs, return an error
	if len(oldState.AccountIDs) == 0 {
		return models.FetchNextAccountsResponse{}, models.ErrMissingFromPayloadInRequest
	}

	newState := walletsState{
		AccountIndex: oldState.AccountIndex,
		AccountID:    oldState.AccountID,
		Skip:         oldState.Skip,
		AccountIDs:   oldState.AccountIDs,
	}

	// If we've processed all accounts, we're done
	if newState.AccountIndex >= len(newState.AccountIDs) {
		payload, err := json.Marshal(newState)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		return models.FetchNextAccountsResponse{
			Accounts: []models.PSPAccount{},
			NewState: payload,
			HasMore:  false,
		}, nil
	}

	// Get the current account ID
	accountID := newState.AccountIDs[newState.AccountIndex]
	newState.AccountID = accountID

	accounts := make([]models.PSPAccount, 0, req.PageSize)
	needMore := false
	hasMore := false

	wallets, hasMoreWallets, err := p.client.GetWallets(ctx, accountID, newState.Skip, req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	for _, wallet := range wallets {
		raw, err := json.Marshal(wallet)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		metadata := map[string]string{
			client.MoovAccountIDMetadataKey: accountID,
			client.MoovWalletIDMetadataKey:  wallet.ID,
		}

		accounts = append(accounts, models.PSPAccount{
			Reference: wallet.ID,
			CreatedAt: time.Now(), // Moov API doesn't provide creation time for wallets
			Type:      models.ACCOUNT_TYPE_WALLET,
			Raw:       raw,
			Metadata:  metadata,
		})
	}

	needMore, hasMore = pagination.ShouldFetchMore(accounts, wallets, req.PageSize)
	if !needMore {
		accounts = accounts[:req.PageSize]
	}

	// Update state for next fetch
	if len(wallets) < req.PageSize {
		// Move to the next account
		newState.AccountIndex++
		newState.Skip = 0
	} else {
		// Continue with the current account
		newState.Skip += len(wallets)
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  hasMore || hasMoreWallets || newState.AccountIndex < len(newState.AccountIDs),
	}, nil
}