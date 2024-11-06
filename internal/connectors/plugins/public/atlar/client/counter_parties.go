package client

import (
	"context"
	"errors"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/get-momo/atlar-v1-go-client/client/counterparties"
	atlar_models "github.com/get-momo/atlar-v1-go-client/models"
)

func (c *client) GetV1CounterpartiesID(ctx context.Context, counterPartyID string) (*counterparties.GetV1CounterpartiesIDOK, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "get_counterparty")

	getCounterpartyParams := counterparties.GetV1CounterpartiesIDParams{
		Context: ctx,
		ID:      counterPartyID,
	}
	counterpartyResponse, err := c.client.Counterparties.GetV1CounterpartiesID(&getCounterpartyParams)
	if err != nil {
		return nil, err
	}

	return counterpartyResponse, nil
}

func (c *client) PostV1CounterParties(ctx context.Context, newExternalBankAccount models.BankAccount) (*counterparties.PostV1CounterpartiesCreated, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "create_counter_party")

	// TODO: make sure an account with that IBAN does not already exist (Atlar API v2 needed, v1 lacks the filters)
	// alternatively we could query the local DB

	createCounterpartyRequest := atlar_models.CreateCounterpartyRequest{
		Name:      ExtractNamespacedMetadataIgnoreEmpty(newExternalBankAccount.Metadata, "owner/name"),
		PartyType: *ExtractNamespacedMetadataIgnoreEmpty(newExternalBankAccount.Metadata, "owner/type"),
		ContactDetails: &atlar_models.ContactDetails{
			Email: *ExtractNamespacedMetadataIgnoreEmpty(newExternalBankAccount.Metadata, "owner/contact/email"),
			Phone: *ExtractNamespacedMetadataIgnoreEmpty(newExternalBankAccount.Metadata, "owner/contact/phone"),
			Address: &atlar_models.Address{
				StreetName:   *ExtractNamespacedMetadataIgnoreEmpty(newExternalBankAccount.Metadata, "owner/contact/address/streetName"),
				StreetNumber: *ExtractNamespacedMetadataIgnoreEmpty(newExternalBankAccount.Metadata, "owner/contact/address/streetNumber"),
				City:         *ExtractNamespacedMetadataIgnoreEmpty(newExternalBankAccount.Metadata, "owner/contact/address/city"),
				PostalCode:   *ExtractNamespacedMetadataIgnoreEmpty(newExternalBankAccount.Metadata, "owner/contact/address/postalCode"),
				Country:      *ExtractNamespacedMetadataIgnoreEmpty(newExternalBankAccount.Metadata, "owner/contact/address/country"),
			},
		},
		ExternalAccounts: []*atlar_models.CreateEmbeddedExternalAccountRequest{
			{
				// ExternalID could cause problems when synchronizing with Accounts[type=external]
				Bank: &atlar_models.UpdatableBank{
					Bic: func() string {
						if newExternalBankAccount.SwiftBicCode == nil {
							return ""
						}
						return *newExternalBankAccount.SwiftBicCode
					}(),
				},
				Identifiers: extractAtlarAccountIdentifiersFromBankAccount(newExternalBankAccount),
			},
		},
	}
	postCounterpartiesParams := counterparties.PostV1CounterpartiesParams{
		Context:      ctx,
		Counterparty: &createCounterpartyRequest,
	}
	postCounterpartiesResponse, err := c.client.Counterparties.PostV1Counterparties(&postCounterpartiesParams)
	if err != nil {
		return nil, err
	}

	if len(postCounterpartiesResponse.Payload.ExternalAccounts) != 1 {
		// should never occur, but when in case it happens it's nice to have an error to search for
		return nil, errors.New("counterparty was not created with exactly one account")
	}

	return postCounterpartiesResponse, nil
}

func extractAtlarAccountIdentifiersFromBankAccount(bankAccount models.BankAccount) []*atlar_models.AccountIdentifier {
	ownerName := bankAccount.Metadata[atlarMetadataSpecNamespace+"owner/name"]
	ibanType := "IBAN"
	accountIdentifiers := []*atlar_models.AccountIdentifier{{
		HolderName: &ownerName,
		Market:     bankAccount.Country,
		Type:       &ibanType,
		Number:     bankAccount.IBAN,
	}}
	for k := range bankAccount.Metadata {
		// check whether the key has format com.atlar.spec/identifier/<market>/<type>
		identifierData, err := metadataToIdentifierData(k, bankAccount.Metadata[k])
		if err != nil {
			// matadata does not describe an identifier
			continue
		}
		if bankAccount.Country != nil && identifierData.Market == *bankAccount.Country && identifierData.Type == "IBAN" {
			// avoid duplicate identifiers
			continue
		}
		accountIdentifiers = append(accountIdentifiers, &atlar_models.AccountIdentifier{
			HolderName: &ownerName,
			Market:     &identifierData.Market,
			Type:       &identifierData.Type,
			Number:     &identifierData.Number,
		})
	}
	return accountIdentifiers
}