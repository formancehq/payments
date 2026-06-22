package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
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
	MetadataHash  string     `json:"metadataHash,omitempty"`
}

func (b *BankAccount) IdempotencyKey() string {
	ik := bankAccountIK{
		ID: b.ID,
	}

	if len(b.RelatedAccounts) > 0 {
		ik.LastAccountID = &b.RelatedAccounts[len(b.RelatedAccounts)-1].AccountID
	}

	if len(b.Metadata) > 0 {
		//hash the metadata in ik.MetadataHash
		ik.MetadataHash = IdempotencyKey(b.Metadata)
	}

	return IdempotencyKey(ik)
}

// Obfuscate masks the IBAN and account number in place, keeping a short
// prefix and suffix for recognisability. Values too short to keep both ends
// are fully masked rather than rejected: these are display-only fields, and
// failing here would surface as a 500 to the caller (and, for the
// forward-to-connector endpoint, only after the side effect already
// succeeded). The error return is kept for interface stability; it is
// currently always nil.
func (a *BankAccount) Obfuscate() error {
	if a.IBAN != nil {
		*a.IBAN = obfuscate(*a.IBAN, 4, 4)
	}

	if a.AccountNumber != nil {
		*a.AccountNumber = obfuscate(*a.AccountNumber, 2, 3)
	}

	return nil
}

// obfuscate keeps the first prefix and last suffix characters of v and masks
// the rest with '*'. If v is too short to keep both ends without overlap, it
// is fully masked.
func obfuscate(v string, prefix, suffix int) string {
	length := len(v)
	if length < prefix+suffix {
		return strings.Repeat("*", length)
	}

	return v[:prefix] + strings.Repeat("*", length-prefix-suffix) + v[length-suffix:]
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
