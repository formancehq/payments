package plaid

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/plaid/plaid-go/v34/plaid"
)

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var from models.BankBridgeFromPayload
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	var baseWebhook client.BaseWebhooks
	if err := json.Unmarshal(from.FromPayload, &baseWebhook); err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	resp, err := p.client.ListAccounts(
		ctx,
		from.PSUBankBridgeConnection.AccessToken.Token,
	)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accounts := make([]models.PSPAccount, 0, len(resp.Accounts))
	for _, account := range resp.Accounts {
		accounts = append(accounts, translatePlaidAccountToPSPAccount(account, from.PSUBankBridgeConnection.ConnectionID))
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
	}, nil
}

func translatePlaidAccountToPSPAccount(account plaid.AccountBase, connectionID string) models.PSPAccount {
	raw, err := json.Marshal(account)
	if err != nil {
		return models.PSPAccount{}
	}

	return models.PSPAccount{
		Reference:    account.AccountId,
		CreatedAt:    time.Now().UTC(),
		Name:         &account.Name,
		DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, account.Balances.GetIsoCurrencyCode())),
		Metadata: map[string]string{
			"accountType":                        string(account.Type),
			models.ObjectConnectionIDMetadataKey: connectionID,
		},
		Raw: raw,
	}
}
