package atlar

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v3/metadata"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/get-momo/atlar-v1-go-client/client/external_accounts"
	atlar_models "github.com/get-momo/atlar-v1-go-client/models"
)

type externalAccountsState struct {
	NextToken string `json:"nextToken"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req connector.FetchNextExternalAccountsRequest) (connector.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}
	}

	var externalAccounts []connector.PSPAccount
	nextToken := oldState.NextToken
	for {
		resp, err := p.client.GetV1ExternalAccounts(ctx, nextToken, int64(req.PageSize))
		if err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}

		externalAccounts, err = p.fillExternalAccounts(ctx, resp, externalAccounts)
		if err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}

		nextToken = resp.Payload.NextToken
		if resp.Payload.NextToken == "" || len(externalAccounts) >= req.PageSize {
			break
		}
	}

	// If token is empty, this is perfect as the next polling task will refetch
	// everything ! And that's what we want since Atlar doesn't provide any
	// filters or sorting options.
	newState := externalAccountsState{
		NextToken: nextToken,
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextExternalAccountsResponse{}, err
	}

	return connector.FetchNextExternalAccountsResponse{
		ExternalAccounts: externalAccounts,
		NewState:         payload,
		HasMore:          nextToken != "",
	}, nil
}

func (p *Plugin) fillExternalAccounts(
	ctx context.Context,
	pagedExternalAccounts *external_accounts.GetV1ExternalAccountsOK,
	accounts []connector.PSPAccount,
) ([]connector.PSPAccount, error) {
	for _, externalAccount := range pagedExternalAccounts.Payload.Items {
		resp, err := p.client.GetV1CounterpartiesID(ctx, externalAccount.CounterpartyID)
		if err != nil {
			return nil, err
		}
		counterparty := resp.Payload

		newAccount, err := externalAccountFromAtlarData(externalAccount, counterparty)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, newAccount)
	}

	return accounts, nil
}

type AtlarExternalAccountAndCounterparty struct {
	ExternalAccount atlar_models.ExternalAccount `json:"externalAccount" yaml:"externalAccount" bson:"externalAccount"`
	Counterparty    atlar_models.Counterparty    `json:"counterparty" yaml:"counterparty" bson:"counterparty"`
}

func externalAccountFromAtlarData(
	externalAccount *atlar_models.ExternalAccount,
	counterparty *atlar_models.Counterparty,
) (connector.PSPAccount, error) {
	raw, err := json.Marshal(AtlarExternalAccountAndCounterparty{ExternalAccount: *externalAccount, Counterparty: *counterparty})
	if err != nil {
		return connector.PSPAccount{}, err
	}

	createdAt, err := ParseAtlarTimestamp(externalAccount.Created)
	if err != nil {
		return connector.PSPAccount{}, fmt.Errorf("failed to parse opening date: %w", err)
	}

	return connector.PSPAccount{
		Reference: externalAccount.ID,
		CreatedAt: createdAt,
		Name:      &counterparty.Name,
		Metadata:  extractExternalAccountAndCounterpartyMetadata(externalAccount, counterparty),
		Raw:       raw,
	}, nil
}

func extractExternalAccountAndCounterpartyMetadata(externalAccount *atlar_models.ExternalAccount, counterparty *atlar_models.Counterparty) metadata.Metadata {
	result := metadata.Metadata{}
	result = result.Merge(computeMetadata("bank/id", externalAccount.Bank.ID))
	result = result.Merge(computeMetadata("bank/name", externalAccount.Bank.Name))
	result = result.Merge(computeMetadata("bank/bic", externalAccount.Bank.Bic))
	result = result.Merge(identifiersToMetadata(externalAccount.Identifiers))
	result = result.Merge(computeMetadata("owner/name", counterparty.Name))
	result = result.Merge(computeMetadata("owner/type", counterparty.PartyType))
	result = result.Merge(computeMetadata("owner/contact/email", counterparty.ContactDetails.Email))
	result = result.Merge(computeMetadata("owner/contact/phone", counterparty.ContactDetails.Phone))
	result = result.Merge(computeMetadata("owner/contact/address/streetName", counterparty.ContactDetails.Address.StreetName))
	result = result.Merge(computeMetadata("owner/contact/address/streetNumber", counterparty.ContactDetails.Address.StreetNumber))
	result = result.Merge(computeMetadata("owner/contact/address/city", counterparty.ContactDetails.Address.City))
	result = result.Merge(computeMetadata("owner/contact/address/postalCode", counterparty.ContactDetails.Address.PostalCode))
	result = result.Merge(computeMetadata("owner/contact/address/country", counterparty.ContactDetails.Address.Country))
	return result
}
