package increase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

type externalAccountsState struct {
	LastID   string          `json:"last_id"`
	Timeline json.RawMessage `json:"timeline"`
}

func (p *Plugin) FetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	var state externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextExternalAccountsResponse{}, fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}

	accounts, nextCursor, hasMore, err := p.client.GetExternalAccounts(ctx, state.LastID, int64(req.PageSize))
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, fmt.Errorf("failed to get external accounts: %w", err)
	}

	pspAccounts := make([]models.PSPAccount, len(accounts))
	for i, account := range accounts {
		raw, err := json.Marshal(account)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, fmt.Errorf("failed to marshal account: %w", err)
		}

		pspAccounts[i] = models.PSPAccount{
			ID:        account.ID,
			Reference: account.ID,
			Type:      models.AccountType(account.Type),
			Status:    models.AccountStatus(account.Status),
			Raw:       raw,
		}
	}

	newState := externalAccountsState{
		LastID: nextCursor,
	}
	newStateBytes, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, fmt.Errorf("failed to marshal new state: %w", err)
	}

	return models.FetchNextExternalAccountsResponse{
		ExternalAccounts: pspAccounts,
		NewState:         newStateBytes,
		HasMore:          hasMore,
	}, nil
}

func (p *Plugin) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	createReq := &client.CreateExternalAccountRequest{
		Name:          req.BankAccount.Name,
		AccountNumber: req.BankAccount.AccountNumber,
		RoutingNumber: req.BankAccount.RoutingNumber,
	}

	account, err := p.client.CreateExternalAccount(ctx, createReq)
	if err != nil {
		return models.CreateBankAccountResponse{}, fmt.Errorf("failed to create external account: %w", err)
	}

	raw, err := json.Marshal(account)
	if err != nil {
		return models.CreateBankAccountResponse{}, fmt.Errorf("failed to marshal account: %w", err)
	}

	return models.CreateBankAccountResponse{
		RelatedAccount: models.PSPAccount{
			ID:        account.ID,
			Reference: account.ID,
			Type:      models.AccountType(account.Type),
			Status:    models.AccountStatus(account.Status),
			Raw:       raw,
		},
	}, nil
}
