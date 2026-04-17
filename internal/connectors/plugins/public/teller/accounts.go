package teller

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/teller/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

type tellerAccountWithBalance struct {
	Account client.Account  `json:"account"`
	Balance *client.Balance `json:"balance"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var from models.OpenBankingForwardedUserFromPayload
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accessToken := from.OpenBankingConnection.AccessToken.Token

	tellerAccounts, err := p.client.ListAccounts(ctx, accessToken)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accounts := make([]models.PSPAccount, 0, len(tellerAccounts))
	for _, account := range tellerAccounts {
		// Fetch balance for each account and embed in Raw
		balance, err := p.client.GetBalance(ctx, accessToken, account.ID)
		if err != nil {
			// Log but don't fail — balance fetch might fail for some account types
			p.logger.Errorf("failed to fetch balance for account %s: %v", account.ID, err)
			balance = nil
		}

		pspAccount := translateTellerAccountToPSPAccount(account, balance, from.PSUID, from.OpenBankingConnection.ConnectionID)
		accounts = append(accounts, pspAccount)
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
	}, nil
}

func translateTellerAccountToPSPAccount(account client.Account, balance *client.Balance, psuID uuid.UUID, connectionID string) models.PSPAccount {
	combined := tellerAccountWithBalance{
		Account: account,
		Balance: balance,
	}
	raw, err := json.Marshal(combined)
	if err != nil {
		return models.PSPAccount{}
	}

	curr := strings.ToUpper(account.Currency)

	return models.PSPAccount{
		Reference:               account.ID,
		CreatedAt:               time.Now().UTC(),
		Name:                    &account.Name,
		DefaultAsset:            pointer.For(curr + "/2"),
		PsuID:                   &psuID,
		OpenBankingConnectionID: &connectionID,
		Metadata: map[string]string{
			"accountType":    account.Type,
			"accountSubtype": account.Subtype,
			"institution":    account.Institution.Name,
			"lastFour":       account.LastFour,
		},
		Raw: raw,
	}
}
