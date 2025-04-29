package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/google/uuid"
)

const (
	bankAccountOwnerNamespace = formanceMetadataSpecNamespace + "owner/"

	// Bank Account metadata
	BankAccountOwnerAddressLine1MetadataKey = bankAccountOwnerNamespace + "addressLine1"
	BankAccountOwnerAddressLine2MetadataKey = bankAccountOwnerNamespace + "addressLine2"
	BankAccountOwnerStreetNameMetadataKey   = bankAccountOwnerNamespace + "streetName"
	BankAccountOwnerStreetNumberMetadataKey = bankAccountOwnerNamespace + "streetNumber"
	BankAccountOwnerCityMetadataKey         = bankAccountOwnerNamespace + "city"
	BankAccountOwnerRegionMetadataKey       = bankAccountOwnerNamespace + "region"
	BankAccountOwnerPostalCodeMetadataKey   = bankAccountOwnerNamespace + "postalCode"
	BankAccountOwnerEmailMetadataKey        = bankAccountOwnerNamespace + "email"
	BankAccountOwnerPhoneNumberMetadataKey  = bankAccountOwnerNamespace + "phoneNumber"

	// Account metadata
	AccountIBANMetadataKey               = bankAccountOwnerNamespace + "iban"
	AccountAccountNumberMetadataKey      = bankAccountOwnerNamespace + "accountNumber"
	AccountBankAccountNameMetadataKey    = bankAccountOwnerNamespace + "name"
	AccountBankAccountCountryMetadataKey = bankAccountOwnerNamespace + "country"
	AccountSwiftBicCodeMetadataKey       = bankAccountOwnerNamespace + "swiftBicCode"
)

type BankAccount struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Name      string    `json:"name"`

	AccountNumber *string `json:"accountNumber"`
	IBAN          *string `json:"iban"`
	SwiftBicCode  *string `json:"swiftBicCode"`
	Country       *string `json:"country"`

	Metadata map[string]string `json:"metadata"`

	RelatedAccounts []BankAccountRelatedAccount `json:"relatedAccounts"`
}

type bankAccountIK struct {
	ID            uuid.UUID  `json:"id"`
	LastAccountID *AccountID `json:"lastAccountID,omitempty"`
}

func (b *BankAccount) IdempotencyKey() string {
	ik := bankAccountIK{
		ID: b.ID,
	}

	if len(b.RelatedAccounts) > 0 {
		ik.LastAccountID = &b.RelatedAccounts[len(b.RelatedAccounts)-1].AccountID
	}

	return IdempotencyKey(ik)
}

func (a *BankAccount) Obfuscate() error {
	if a.IBAN != nil {
		length := len(*a.IBAN)
		if length < 8 {
			return errors.New("IBAN is not valid")
		}

		*a.IBAN = (*a.IBAN)[:4] + strings.Repeat("*", length-8) + (*a.IBAN)[length-4:]
	}

	if a.AccountNumber != nil {
		length := len(*a.AccountNumber)
		if length < 5 {
			return errors.New("Account number is not valid")
		}

		*a.AccountNumber = (*a.AccountNumber)[:2] + strings.Repeat("*", length-5) + (*a.AccountNumber)[length-3:]
	}

	return nil
}

func FillBankAccountDetailsToAccountMetadata(account *Account, bankAccount *BankAccount) {
	if account.Metadata == nil {
		account.Metadata = make(map[string]string)
	}

	account.Metadata[BankAccountOwnerAddressLine1MetadataKey] = bankAccount.Metadata[BankAccountOwnerAddressLine1MetadataKey]
	account.Metadata[BankAccountOwnerAddressLine2MetadataKey] = bankAccount.Metadata[BankAccountOwnerAddressLine2MetadataKey]
	account.Metadata[BankAccountOwnerCityMetadataKey] = bankAccount.Metadata[BankAccountOwnerCityMetadataKey]
	account.Metadata[BankAccountOwnerRegionMetadataKey] = bankAccount.Metadata[BankAccountOwnerRegionMetadataKey]
	account.Metadata[BankAccountOwnerPostalCodeMetadataKey] = bankAccount.Metadata[BankAccountOwnerPostalCodeMetadataKey]

	if bankAccount.AccountNumber != nil {
		account.Metadata[AccountAccountNumberMetadataKey] = *bankAccount.AccountNumber
	}

	if bankAccount.IBAN != nil {
		account.Metadata[AccountIBANMetadataKey] = *bankAccount.IBAN
	}

	if bankAccount.SwiftBicCode != nil {
		account.Metadata[AccountSwiftBicCodeMetadataKey] = *bankAccount.SwiftBicCode
	}

	if bankAccount.Country != nil {
		account.Metadata[AccountBankAccountCountryMetadataKey] = *bankAccount.Country
	}

	account.Metadata[AccountBankAccountNameMetadataKey] = bankAccount.Name
}

func FillBankAccountMetadataWithPaymentServiceUserInfo(ba *BankAccount, psu *PaymentServiceUser) {
	if psu.Address != nil {
		var addressLine1 *string
		switch {
		case psu.Address.StreetNumber != nil && psu.Address.StreetName != nil:
			addressLine1 = pointer.For(fmt.Sprintf("%s %s", *psu.Address.StreetNumber, *psu.Address.StreetName))
		case psu.Address.StreetName != nil:
			addressLine1 = psu.Address.StreetName
		}

		fillMetadata(ba.Metadata, BankAccountOwnerAddressLine1MetadataKey, addressLine1)
		fillMetadata(ba.Metadata, BankAccountOwnerStreetNameMetadataKey, psu.Address.StreetName)
		fillMetadata(ba.Metadata, BankAccountOwnerStreetNumberMetadataKey, psu.Address.StreetNumber)
		fillMetadata(ba.Metadata, BankAccountOwnerCityMetadataKey, psu.Address.City)
		fillMetadata(ba.Metadata, BankAccountOwnerRegionMetadataKey, psu.Address.Region)
		fillMetadata(ba.Metadata, BankAccountOwnerPostalCodeMetadataKey, psu.Address.PostalCode)
	}

	if psu.ContactDetails != nil {
		fillMetadata(ba.Metadata, BankAccountOwnerEmailMetadataKey, psu.ContactDetails.Email)
		fillMetadata(ba.Metadata, BankAccountOwnerPhoneNumberMetadataKey, psu.ContactDetails.PhoneNumber)
	}
}

func fillMetadata(metadata map[string]string, key string, value *string) {
	if value != nil {
		metadata[key] = *value
	}
}
