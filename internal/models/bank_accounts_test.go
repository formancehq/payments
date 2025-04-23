package models_test

import (
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBankAccountIdempotencyKey(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	bankAccount := models.BankAccount{
		ID: id,
	}

	key := bankAccount.IdempotencyKey()
	assert.Equal(t, "a91e0ca356b7581fec04c398da35574f7db6fb40", key)
}

func TestBankAccountObfuscate(t *testing.T) {
	t.Parallel()

	t.Run("valid IBAN and account number", func(t *testing.T) {
		t.Parallel()

		iban := "DE89370400440532013000"
		accountNumber := "12345678901"
		bankAccount := models.BankAccount{
			IBAN:          &iban,
			AccountNumber: &accountNumber,
		}

		err := bankAccount.Obfuscate()
		require.NoError(t, err)

		assert.Equal(t, "DE89**************3000", *bankAccount.IBAN)

		assert.Equal(t, "12******901", *bankAccount.AccountNumber)
	})

	t.Run("invalid IBAN", func(t *testing.T) {
		t.Parallel()

		iban := "DE89"
		bankAccount := models.BankAccount{
			IBAN: &iban,
		}

		err := bankAccount.Obfuscate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "IBAN is not valid")
	})

	t.Run("invalid account number", func(t *testing.T) {
		t.Parallel()

		accountNumber := "123"
		bankAccount := models.BankAccount{
			AccountNumber: &accountNumber,
		}

		err := bankAccount.Obfuscate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Account number is not valid")
	})

	t.Run("nil IBAN and account number", func(t *testing.T) {
		t.Parallel()

		bankAccount := models.BankAccount{}

		err := bankAccount.Obfuscate()
		require.NoError(t, err)
	})
}

func TestFillBankAccountDetailsToAccountMetadata(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.New(),
	}

	t.Run("with all bank account fields", func(t *testing.T) {
		t.Parallel()

		iban := "DE89370400440532013000"
		accountNumber := "12345678901"
		swiftBicCode := "DEUTDEFF"
		country := "DE"

		bankAccount := &models.BankAccount{
			ID:            uuid.New(),
			CreatedAt:     now,
			Name:          "Test Bank Account",
			AccountNumber: &accountNumber,
			IBAN:          &iban,
			SwiftBicCode:  &swiftBicCode,
			Country:       &country,
			Metadata: map[string]string{
				models.BankAccountOwnerAddressLine1MetadataKey: "123 Main St",
				models.BankAccountOwnerAddressLine2MetadataKey: "Apt 4B",
				models.BankAccountOwnerCityMetadataKey:         "Berlin",
				models.BankAccountOwnerRegionMetadataKey:       "Berlin",
				models.BankAccountOwnerPostalCodeMetadataKey:   "10115",
			},
		}

		account := &models.Account{
			ID: models.AccountID{
				Reference:   "acc123",
				ConnectorID: connectorID,
			},
		}

		models.FillBankAccountDetailsToAccountMetadata(account, bankAccount)

		assert.Equal(t, "123 Main St", account.Metadata[models.BankAccountOwnerAddressLine1MetadataKey])
		assert.Equal(t, "Apt 4B", account.Metadata[models.BankAccountOwnerAddressLine2MetadataKey])
		assert.Equal(t, "Berlin", account.Metadata[models.BankAccountOwnerCityMetadataKey])
		assert.Equal(t, "Berlin", account.Metadata[models.BankAccountOwnerRegionMetadataKey])
		assert.Equal(t, "10115", account.Metadata[models.BankAccountOwnerPostalCodeMetadataKey])
		assert.Equal(t, accountNumber, account.Metadata[models.BankAccountAccountNumberMetadataKey])
		assert.Equal(t, iban, account.Metadata[models.BankAccountIBANMetadataKey])
		assert.Equal(t, swiftBicCode, account.Metadata[models.BankAccountSwiftBicCodeMetadataKey])
		assert.Equal(t, country, account.Metadata[models.BankAccountCountryMetadataKey])
		assert.Equal(t, "Test Bank Account", account.Metadata[models.BankAccountNameMetadataKey])
	})

	t.Run("with nil account metadata", func(t *testing.T) {
		t.Parallel()

		bankAccount := &models.BankAccount{
			ID:        uuid.New(),
			CreatedAt: now,
			Name:      "Test Bank Account",
			Metadata:  map[string]string{},
		}

		account := &models.Account{
			ID: models.AccountID{
				Reference:   "acc123",
				ConnectorID: connectorID,
			},
			Metadata: nil, // Explicitly set to nil to test initialization
		}

		models.FillBankAccountDetailsToAccountMetadata(account, bankAccount)

		assert.NotNil(t, account.Metadata)
		assert.Equal(t, "Test Bank Account", account.Metadata[models.BankAccountNameMetadataKey])
	})

	t.Run("with nil optional fields", func(t *testing.T) {
		t.Parallel()

		bankAccount := &models.BankAccount{
			ID:        uuid.New(),
			CreatedAt: now,
			Name:      "Test Bank Account",
			Metadata:  map[string]string{},
		}

		account := &models.Account{
			ID: models.AccountID{
				Reference:   "acc123",
				ConnectorID: connectorID,
			},
			Metadata: map[string]string{},
		}

		models.FillBankAccountDetailsToAccountMetadata(account, bankAccount)

		assert.Equal(t, "Test Bank Account", account.Metadata[models.BankAccountNameMetadataKey])
		_, hasAccountNumber := account.Metadata[models.BankAccountAccountNumberMetadataKey]
		assert.False(t, hasAccountNumber)
		_, hasIBAN := account.Metadata[models.BankAccountIBANMetadataKey]
		assert.False(t, hasIBAN)
		_, hasSwiftBicCode := account.Metadata[models.BankAccountSwiftBicCodeMetadataKey]
		assert.False(t, hasSwiftBicCode)
		_, hasCountry := account.Metadata[models.BankAccountCountryMetadataKey]
		assert.False(t, hasCountry)
	})
}
