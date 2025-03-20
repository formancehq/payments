package mangopay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createBankAccountFromBankAccountModels(
	ctx context.Context,
	ba *models.BankAccount,
) (models.CreateBankAccountResponse, error) {
	userID := models.ExtractNamespacedMetadata(ba.Metadata, client.MangopayUserIDMetadataKey)
	if userID == "" {
		return models.CreateBankAccountResponse{}, models.NewConnectorMetadataError(client.MangopayUserIDMetadataKey)
	}

	ownerAddress := client.OwnerAddress{
		AddressLine1: models.ExtractNamespacedMetadata(ba.Metadata, models.BankAccountOwnerAddressLine1MetadataKey),
		AddressLine2: models.ExtractNamespacedMetadata(ba.Metadata, models.BankAccountOwnerAddressLine2MetadataKey),
		City:         models.ExtractNamespacedMetadata(ba.Metadata, models.BankAccountOwnerCityMetadataKey),
		Region:       models.ExtractNamespacedMetadata(ba.Metadata, models.BankAccountOwnerRegionMetadataKey),
		PostalCode:   models.ExtractNamespacedMetadata(ba.Metadata, models.BankAccountOwnerPostalCodeMetadataKey),
		Country: func() string {
			if ba.Country == nil {
				return ""
			}
			return *ba.Country
		}(),
	}

	return p.createBankAccount(
		ctx,
		userID,
		ba.Name,
		ownerAddress,
		ba.IBAN,
		ba.AccountNumber,
		ba.SwiftBicCode,
		ba.Metadata,
	)
}

func (p *Plugin) createBankAccountFromCounterPartyModels(
	ctx context.Context,
	cp *models.PSPCounterParty,
) (models.CreateBankAccountResponse, error) {
	userID := models.ExtractNamespacedMetadata(cp.Metadata, client.MangopayUserIDMetadataKey)
	if userID == "" {
		return models.CreateBankAccountResponse{}, models.NewConnectorMetadataError(client.MangopayUserIDMetadataKey)
	}

	addressLine := ""
	city := ""
	region := ""
	postalCode := ""
	country := ""
	if cp.Address != nil {
		addressLine = fmt.Sprintf("%s %s", cp.Address.StreetNumber, cp.Address.StreetName)
		city = cp.Address.City
		region = cp.Address.Region
		postalCode = cp.Address.PostalCode
		country = cp.Address.Country
	}

	ownerAddress := client.OwnerAddress{
		AddressLine1: addressLine,
		City:         city,
		Region:       region,
		PostalCode:   postalCode,
		Country:      country,
	}

	var iban *string
	var accountNumber *string
	var switfBicCode *string
	if cp.BankAccount != nil {
		iban = cp.BankAccount.IBAN
		accountNumber = cp.BankAccount.AccountNumber
		switfBicCode = cp.BankAccount.SwiftBicCode
	}

	return p.createBankAccount(
		ctx,
		userID,
		cp.Name,
		ownerAddress,
		iban,
		accountNumber,
		switfBicCode,
		cp.Metadata,
	)
}

func (p *Plugin) createBankAccount(
	ctx context.Context,
	userID string,
	ownerName string,
	ownerAddress client.OwnerAddress,
	iban *string,
	accountNumber *string,
	swiftBicCode *string,
	metadata map[string]string,
) (models.CreateBankAccountResponse, error) {

	var mangopayBankAccount *client.BankAccount
	if iban != nil {
		req := &client.CreateIBANBankAccountRequest{
			OwnerName:    ownerName,
			OwnerAddress: &ownerAddress,
			IBAN:         *iban,
			BIC: func() string {
				if swiftBicCode == nil {
					return ""
				}
				return *swiftBicCode
			}(),
			Tag: models.ExtractNamespacedMetadata(metadata, client.MangopayTagMetadataKey),
		}

		var err error
		mangopayBankAccount, err = p.client.CreateIBANBankAccount(ctx, userID, req)
		if err != nil {
			return models.CreateBankAccountResponse{}, fmt.Errorf("%w: %w", models.ErrFailedAccountCreation, err)
		}
	} else {
		switch ownerAddress.Country {
		case "US":
			if accountNumber == nil {
				return models.CreateBankAccountResponse{}, models.ErrMissingAccountInRequest
			}

			req := &client.CreateUSBankAccountRequest{
				OwnerName:          ownerName,
				OwnerAddress:       &ownerAddress,
				AccountNumber:      *accountNumber,
				ABA:                models.ExtractNamespacedMetadata(metadata, client.MangopayABAMetadataKey),
				DepositAccountType: models.ExtractNamespacedMetadata(metadata, client.MangopayDepositAccountTypeMetadataKey),
				Tag:                models.ExtractNamespacedMetadata(metadata, client.MangopayTagMetadataKey),
			}

			var err error
			mangopayBankAccount, err = p.client.CreateUSBankAccount(ctx, userID, req)
			if err != nil {
				return models.CreateBankAccountResponse{}, fmt.Errorf("%w: %w", models.ErrFailedAccountCreation, err)
			}

		case "CA":
			if accountNumber == nil {
				return models.CreateBankAccountResponse{}, models.ErrMissingAccountInRequest
			}
			req := &client.CreateCABankAccountRequest{
				OwnerName:         ownerName,
				OwnerAddress:      &ownerAddress,
				AccountNumber:     *accountNumber,
				InstitutionNumber: models.ExtractNamespacedMetadata(metadata, client.MangopayInstitutionNumberMetadataKey),
				BranchCode:        models.ExtractNamespacedMetadata(metadata, client.MangopayBranchCodeMetadataKey),
				BankName:          models.ExtractNamespacedMetadata(metadata, client.MangopayBankNameMetadataKey),
				Tag:               models.ExtractNamespacedMetadata(metadata, client.MangopayTagMetadataKey),
			}

			var err error
			mangopayBankAccount, err = p.client.CreateCABankAccount(ctx, userID, req)
			if err != nil {
				return models.CreateBankAccountResponse{}, fmt.Errorf("%w: %w", models.ErrFailedAccountCreation, err)
			}

		case "GB":
			if accountNumber == nil {
				return models.CreateBankAccountResponse{}, models.ErrMissingAccountInRequest
			}

			req := &client.CreateGBBankAccountRequest{
				OwnerName:     ownerName,
				OwnerAddress:  &ownerAddress,
				AccountNumber: *accountNumber,
				SortCode:      models.ExtractNamespacedMetadata(metadata, client.MangopaySortCodeMetadataKey),
				Tag:           models.ExtractNamespacedMetadata(metadata, client.MangopayTagMetadataKey),
			}

			var err error
			mangopayBankAccount, err = p.client.CreateGBBankAccount(ctx, userID, req)
			if err != nil {
				return models.CreateBankAccountResponse{}, fmt.Errorf("%w: %w", models.ErrFailedAccountCreation, err)
			}

		default:
			if accountNumber == nil {
				return models.CreateBankAccountResponse{}, models.ErrMissingAccountInRequest
			}

			req := &client.CreateOtherBankAccountRequest{
				OwnerName:     ownerName,
				OwnerAddress:  &ownerAddress,
				AccountNumber: *accountNumber,
				BIC: func() string {
					if swiftBicCode == nil {
						return ""
					}
					return *swiftBicCode
				}(),
				Country: ownerAddress.Country,
				Tag:     models.ExtractNamespacedMetadata(metadata, client.MangopayTagMetadataKey),
			}

			var err error
			mangopayBankAccount, err = p.client.CreateOtherBankAccount(ctx, userID, req)
			if err != nil {
				return models.CreateBankAccountResponse{}, fmt.Errorf("%w: %w", models.ErrFailedAccountCreation, err)
			}
		}
	}

	var account models.PSPAccount
	if mangopayBankAccount != nil {
		raw, err := json.Marshal(mangopayBankAccount)
		if err != nil {
			return models.CreateBankAccountResponse{}, err
		}

		account = models.PSPAccount{
			Reference: mangopayBankAccount.ID,
			CreatedAt: time.Unix(mangopayBankAccount.CreationDate, 0),
			Name:      &mangopayBankAccount.OwnerName,
			Metadata: map[string]string{
				"user_id": userID,
			},
			Raw: raw,
		}

	}

	return models.CreateBankAccountResponse{
		RelatedAccount: account,
	}, nil
}
