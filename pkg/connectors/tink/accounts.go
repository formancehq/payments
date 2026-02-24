package tink

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/pkg/connectors/tink/client"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) fetchNextAccounts(ctx context.Context, req connector.FetchNextAccountsRequest) (connector.FetchNextAccountsResponse, error) {
	var from connector.OpenBankingForwardedUserFromPayload
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	var webhook fetchNextDataRequest
	if err := json.Unmarshal(from.FromPayload, &webhook); err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	account, err := p.client.GetAccount(ctx, webhook.ExternalUserID, webhook.AccountID)
	if err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	accounts := make([]connector.PSPAccount, 0, 1)
	accounts, err = toPSPAccounts(accounts, []client.Account{account}, from)
	if err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	return connector.FetchNextAccountsResponse{
		Accounts: accounts,
	}, nil
}

func toPSPAccounts(
	accounts []connector.PSPAccount,
	pagedAccounts []client.Account,
	from connector.OpenBankingForwardedUserFromPayload,
) ([]connector.PSPAccount, error) {
	for _, account := range pagedAccounts {
		raw, err := json.Marshal(account)
		if err != nil {
			return accounts, err
		}

		acc := connector.PSPAccount{
			Reference: account.ID,
			CreatedAt: time.Now().UTC(),
			Name:      &account.Name,
			Metadata:  make(map[string]string),
			PsuID:     &from.PSUID,
			Raw:       raw,
		}

		// Note -- right now Tink doesn't send us the ConnectionID so we can't save it
		if from.OpenBankingConnection != nil {
			acc.OpenBankingConnectionID = &from.OpenBankingConnection.ConnectionID
		}

		accounts = append(accounts, acc)
	}

	return accounts, nil
}
