package plaid

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/pkg/connectors/plaid/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/google/uuid"
	"github.com/plaid/plaid-go/v34/plaid"
)

func (p *Plugin) fetchNextAccounts(ctx context.Context, req connector.FetchNextAccountsRequest) (connector.FetchNextAccountsResponse, error) {
	var from connector.OpenBankingForwardedUserFromPayload
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	var baseWebhook client.BaseWebhooks
	if err := json.Unmarshal(from.FromPayload, &baseWebhook); err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	resp, err := p.client.ListAccounts(
		ctx,
		from.OpenBankingConnection.AccessToken.Token,
	)
	if err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	accounts := make([]connector.PSPAccount, 0, len(resp.Accounts))
	for _, account := range resp.Accounts {
		accounts = append(accounts, translatePlaidAccountToPSPAccount(account, from.PSUID, from.OpenBankingConnection.ConnectionID))
	}

	return connector.FetchNextAccountsResponse{
		Accounts: accounts,
	}, nil
}

func translatePlaidAccountToPSPAccount(account plaid.AccountBase, psuID uuid.UUID, connectionID string) connector.PSPAccount {
	raw, err := json.Marshal(account)
	if err != nil {
		return connector.PSPAccount{}
	}

	return connector.PSPAccount{
		Reference:               account.AccountId,
		CreatedAt:               time.Now().UTC(),
		Name:                    &account.Name,
		DefaultAsset:            pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, account.Balances.GetIsoCurrencyCode())),
		PsuID:                   &psuID,
		OpenBankingConnectionID: &connectionID,
		Metadata: map[string]string{
			"accountType": string(account.Type),
		},
		Raw: raw,
	}
}
