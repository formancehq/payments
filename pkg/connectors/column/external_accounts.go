package column

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/formancehq/payments/pkg/connectors/column/client"
	"github.com/formancehq/payments/pkg/connector"
)

type externalAccountsState struct {
	LastIDCreated string `json:"lastIDCreated"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req connector.FetchNextExternalAccountsRequest) (connector.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}
	}

	newState := externalAccountsState{
		LastIDCreated: oldState.LastIDCreated,
	}

	accounts := make([]connector.PSPAccount, 0, req.PageSize)
	pagedRecipients, hasMore, err := p.client.GetCounterparties(ctx, oldState.LastIDCreated, req.PageSize)
	if err != nil {
		return connector.FetchNextExternalAccountsResponse{}, err
	}

	accounts, err = p.fillExternalAccounts(pagedRecipients, accounts, req.PageSize)
	if err != nil {
		return connector.FetchNextExternalAccountsResponse{}, err
	}

	if len(accounts) > 0 {
		newState.LastIDCreated = accounts[len(accounts)-1].Reference
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextExternalAccountsResponse{}, err
	}

	return connector.FetchNextExternalAccountsResponse{
		ExternalAccounts: accounts,
		NewState:         payload,
		HasMore:          hasMore,
	}, nil
}

func (p *Plugin) fillExternalAccounts(
	pagedRecipients []*client.Counterparties,
	accounts []connector.PSPAccount,
	pageSize int,
) ([]connector.PSPAccount, error) {
	for _, account := range pagedRecipients {
		if len(accounts) > pageSize {
			break
		}

		createdTime, err := time.Parse(time.RFC3339, account.CreatedAt)
		if err != nil {
			return nil, err
		}

		raw, err := json.Marshal(account)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, connector.PSPAccount{
			Reference: account.ID,
			CreatedAt: createdTime,
			Name:      &account.Name,
			Raw:       raw,
			Metadata: map[string]string{
				client.ColumnTypeMetadataKey:                 account.AccountType,
				client.ColumnAccountNumberMetadataKey:        account.AccountNumber,
				client.ColumnAddressCityMetadataKey:          account.Address.City,
				client.ColumnAddressCountryCodeMetadataKey:   account.Address.CountryCode,
				client.ColumnAddressLine1MetadataKey:         account.Address.Line1,
				client.ColumnAddressLine2MetadataKey:         account.Address.Line2,
				client.ColumnAddressPostalCodeMetadataKey:    account.Address.PostalCode,
				client.ColumnAddressStateMetadataKey:         account.Address.State,
				client.ColumnDescriptionMetadataKey:          account.Description,
				client.ColumnEmailMetadataKey:                account.Email,
				client.ColumnIsColumnAccountMetadataKey:      strconv.FormatBool(account.IsColumnAccount),
				client.ColumnLegalIDMetadataKey:              account.LegalID,
				client.ColumnLegalTypeMetadataKey:            account.LegalType,
				client.ColumnLocalAccountNumberMetadataKey:   account.LocalAccountNumber,
				client.ColumnLocalBankCodeMetadataKey:        account.LocalBankCode,
				client.ColumnLocalBankCountryCodeMetadataKey: account.LocalBankCountryCode,
				client.ColumnLocalBankNameMetadataKey:        account.LocalBankName,
				client.ColumnPhoneMetadataKey:                account.Phone,
				client.ColumnRoutingNumberTypeMetadataKey:    account.RoutingNumberType,
				client.ColumnUpdatedAtMetadataKey:            account.UpdatedAt,
				client.ColumnRoutingNumberMetadataKey:        account.RoutingNumber,
				client.ColumnWireDrawdownAllowedMetadataKey:  strconv.FormatBool(account.WireDrawdownAllowed),
			},
		})
	}
	return accounts, nil
}
