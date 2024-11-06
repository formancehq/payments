package atlar

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v2/metadata"
	"github.com/formancehq/payments/internal/models"
	atlar_models "github.com/get-momo/atlar-v1-go-client/models"
)

type AtlarExternalAccountAndCounterparty struct {
	ExternalAccount atlar_models.ExternalAccount `json:"externalAccount" yaml:"externalAccount" bson:"externalAccount"`
	Counterparty    atlar_models.Counterparty    `json:"counterparty" yaml:"counterparty" bson:"counterparty"`
}

func ExternalAccountFromAtlarData(
	externalAccount *atlar_models.ExternalAccount,
	counterparty *atlar_models.Counterparty,
) (models.PSPAccount, error) {
	raw, err := json.Marshal(AtlarExternalAccountAndCounterparty{ExternalAccount: *externalAccount, Counterparty: *counterparty})
	if err != nil {
		return models.PSPAccount{}, err
	}

	createdAt, err := ParseAtlarTimestamp(externalAccount.Created)
	if err != nil {
		return models.PSPAccount{}, fmt.Errorf("failed to parse opening date: %w", err)
	}

	return models.PSPAccount{
		Reference: externalAccount.ID,
		CreatedAt: createdAt,
		Name:      &counterparty.Name,
		Metadata:  extractExternalAccountAndCounterpartyMetadata(externalAccount, counterparty),
		Raw:       raw,
	}, nil
}

func ExtractAccountMetadata(account *atlar_models.Account, bank *atlar_models.ThirdParty) metadata.Metadata {
	result := metadata.Metadata{}
	result = result.Merge(computeMetadataBool("fictive", account.Fictive))
	result = result.Merge(computeMetadata("bank/id", bank.ID))
	result = result.Merge(computeMetadata("bank/name", bank.Name))
	result = result.Merge(computeMetadata("bank/bic", account.Bank.Bic))
	result = result.Merge(IdentifiersToMetadata(account.Identifiers))
	result = result.Merge(computeMetadata("alias", account.Alias))
	result = result.Merge(computeMetadata("owner/name", account.Owner.Name))
	return result
}

func IdentifiersToMetadata(identifiers []*atlar_models.AccountIdentifier) metadata.Metadata {
	result := metadata.Metadata{}
	for _, i := range identifiers {
		result = result.Merge(computeMetadata(
			fmt.Sprintf("identifier/%s/%s", *i.Market, *i.Type),
			*i.Number,
		))
		if *i.Type == "IBAN" {
			result = result.Merge(computeMetadata(
				fmt.Sprintf("identifier/%s", *i.Type),
				*i.Number,
			))
		}
	}
	return result
}

func extractExternalAccountAndCounterpartyMetadata(externalAccount *atlar_models.ExternalAccount, counterparty *atlar_models.Counterparty) metadata.Metadata {
	result := metadata.Metadata{}
	result = result.Merge(computeMetadata("bank/id", externalAccount.Bank.ID))
	result = result.Merge(computeMetadata("bank/name", externalAccount.Bank.Name))
	result = result.Merge(computeMetadata("bank/bic", externalAccount.Bank.Bic))
	result = result.Merge(IdentifiersToMetadata(externalAccount.Identifiers))
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
