package tink

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var from models.BankBridgeFromPayload
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	var webhook fetchNextDataRequest
	if err := json.Unmarshal(from.FromPayload, &webhook); err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	account, err := p.client.GetAccount(ctx, webhook.ExternalUserID, webhook.AccountID)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accounts := make([]models.PSPAccount, 0, 1)
	accounts, err = toPSPAccounts(accounts, []client.Account{account}, from)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
	}, nil
}

func toPSPAccounts(
	accounts []models.PSPAccount,
	pagedAccounts []client.Account,
	from models.BankBridgeFromPayload,
) ([]models.PSPAccount, error) {
	for _, account := range pagedAccounts {
		raw, err := json.Marshal(account)
		if err != nil {
			return accounts, err
		}

		acc := models.PSPAccount{
			Reference: account.ID,
			CreatedAt: time.Now().UTC(),
			Name:      &account.Name,
			Metadata:  make(map[string]string),
			Raw:       raw,
		}

		if from.PSUBankBridge != nil {
			acc.Metadata[models.ObjectPSUIDMetadataKey] = from.PSUBankBridge.PsuID.String()
		}

		if from.PSUBankBridgeConnection != nil {
			acc.Metadata[models.ObjectConnectionIDMetadataKey] = from.PSUBankBridgeConnection.ConnectionID
		}

		accounts = append(accounts, acc)
	}

	return accounts, nil
}
